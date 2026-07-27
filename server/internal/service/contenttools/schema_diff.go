package contenttools

import (
	"encoding/json"
	"fmt"
	"sort"
)

// SchemaDiffFinding names a path whose change requires a particular bump.
type SchemaDiffFinding struct {
	Path string
	Kind BumpKind
	Note string
}

// ClassifySchemaDiff compares two JSON Schema documents and returns the minimum
// required version bump plus offending paths (FR-3 / FR-4 / AC-2).
//
// Rules:
//   - additive optional field → minor
//   - new required field, removed field, narrowed type, changed enum member → major
//   - documentation / title / description-only → patch (treated as none for enforcement
//     when version is unchanged; otherwise patch is sufficient)
func ClassifySchemaDiff(oldSchema, newSchema json.RawMessage) (BumpKind, []SchemaDiffFinding, error) {
	var oldDoc, newDoc any
	if len(oldSchema) == 0 {
		oldDoc = map[string]any{}
	} else if err := json.Unmarshal(oldSchema, &oldDoc); err != nil {
		return BumpNone, nil, fmt.Errorf("old schema: %w", err)
	}
	if len(newSchema) == 0 {
		newDoc = map[string]any{}
	} else if err := json.Unmarshal(newSchema, &newDoc); err != nil {
		return BumpNone, nil, fmt.Errorf("new schema: %w", err)
	}
	findings := diffSchemaNode("$", oldDoc, newDoc, nil)
	max := BumpNone
	for _, f := range findings {
		if RequiredBumpRank(f.Kind) > RequiredBumpRank(max) {
			max = f.Kind
		}
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].Path < findings[j].Path })
	return max, findings, nil
}

// AssertVersionCoversSchemaDiff fails when the declared bump is weaker than required.
func AssertVersionCoversSchemaDiff(fromVer, toVer string, oldSchema, newSchema json.RawMessage) error {
	declared, err := ClassifyVersionBump(fromVer, toVer)
	if err != nil {
		return err
	}
	required, findings, err := ClassifySchemaDiff(oldSchema, newSchema)
	if err != nil {
		return err
	}
	if RequiredBumpRank(declared) >= RequiredBumpRank(required) {
		return nil
	}
	paths := make([]string, 0, len(findings))
	for _, f := range findings {
		if RequiredBumpRank(f.Kind) >= RequiredBumpRank(required) {
			paths = append(paths, fmt.Sprintf("%s (%s: %s)", f.Path, f.Kind, f.Note))
		}
	}
	return fmt.Errorf("schema diff requires %s bump but version %s → %s is only %s; offending: %s",
		required, fromVer, toVer, declared, stringsJoin(paths, "; "))
}

func stringsJoin(parts []string, sep string) string {
	if len(parts) == 0 {
		return "(none)"
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += sep + parts[i]
	}
	return out
}

func asObject(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func diffSchemaNode(path string, oldV, newV any, out []SchemaDiffFinding) []SchemaDiffFinding {
	oldObj, oldIsObj := asObject(oldV)
	newObj, newIsObj := asObject(newV)
	if !oldIsObj || !newIsObj {
		if !jsonEqual(oldV, newV) {
			out = append(out, SchemaDiffFinding{Path: path, Kind: BumpMajor, Note: "type or value changed"})
		}
		return out
	}

	// Type narrowing / change.
	oldType, _ := oldObj["type"].(string)
	newType, _ := newObj["type"].(string)
	if oldType != "" && newType != "" && oldType != newType {
		out = append(out, SchemaDiffFinding{Path: path + ".type", Kind: BumpMajor, Note: fmt.Sprintf("%s → %s", oldType, newType)})
	}

	// Enum member changes.
	oldEnum := asAnySlice(oldObj["enum"])
	newEnum := asAnySlice(newObj["enum"])
	if oldEnum != nil || newEnum != nil {
		out = diffEnum(path+".enum", oldEnum, newEnum, out)
	}

	// Properties.
	oldProps, _ := asObject(oldObj["properties"])
	newProps, _ := asObject(newObj["properties"])
	if oldProps == nil {
		oldProps = map[string]any{}
	}
	if newProps == nil {
		newProps = map[string]any{}
	}
	oldReq := requiredSet(oldObj["required"])
	newReq := requiredSet(newObj["required"])

	for name, oldProp := range oldProps {
		p := path + ".properties." + name
		newProp, exists := newProps[name]
		if !exists {
			out = append(out, SchemaDiffFinding{Path: p, Kind: BumpMajor, Note: "field removed"})
			continue
		}
		out = diffSchemaNode(p, oldProp, newProp, out)
		if !oldReq[name] && newReq[name] {
			out = append(out, SchemaDiffFinding{Path: p, Kind: BumpMajor, Note: "became required"})
		}
	}
	for name, newProp := range newProps {
		if _, exists := oldProps[name]; exists {
			continue
		}
		p := path + ".properties." + name
		if newReq[name] {
			out = append(out, SchemaDiffFinding{Path: p, Kind: BumpMajor, Note: "new required field"})
		} else {
			out = append(out, SchemaDiffFinding{Path: p, Kind: BumpMinor, Note: "additive optional field"})
		}
		_ = newProp
	}

	// Doc-only keys: if only title/description changed at this node and nothing else, patch.
	if onlyDocChanged(oldObj, newObj) {
		out = append(out, SchemaDiffFinding{Path: path, Kind: BumpPatch, Note: "documentation-only change"})
	}

	return out
}

func diffEnum(path string, oldEnum, newEnum []any, out []SchemaDiffFinding) []SchemaDiffFinding {
	oldSet := map[string]struct{}{}
	for _, v := range oldEnum {
		oldSet[jsonKey(v)] = struct{}{}
	}
	newSet := map[string]struct{}{}
	for _, v := range newEnum {
		newSet[jsonKey(v)] = struct{}{}
	}
	for k := range oldSet {
		if _, ok := newSet[k]; !ok {
			out = append(out, SchemaDiffFinding{Path: path, Kind: BumpMajor, Note: "enum member removed: " + k})
		}
	}
	for k := range newSet {
		if _, ok := oldSet[k]; !ok {
			// Adding enum members is typically non-breaking for writers; treat as minor.
			out = append(out, SchemaDiffFinding{Path: path, Kind: BumpMinor, Note: "enum member added: " + k})
		}
	}
	return out
}

func requiredSet(v any) map[string]bool {
	out := map[string]bool{}
	for _, item := range asAnySlice(v) {
		if s, ok := item.(string); ok {
			out[s] = true
		}
	}
	return out
}

func asAnySlice(v any) []any {
	if v == nil {
		return nil
	}
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}

func jsonEqual(a, b any) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}

func jsonKey(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

func onlyDocChanged(oldObj, newObj map[string]any) bool {
	docKeys := map[string]struct{}{"title": {}, "description": {}, "$comment": {}}
	oldCopy := map[string]any{}
	newCopy := map[string]any{}
	docDiff := false
	for k, v := range oldObj {
		if _, ok := docKeys[k]; ok {
			if !jsonEqual(v, newObj[k]) {
				docDiff = true
			}
			continue
		}
		oldCopy[k] = v
	}
	for k, v := range newObj {
		if _, ok := docKeys[k]; ok {
			if !jsonEqual(v, oldObj[k]) {
				docDiff = true
			}
			continue
		}
		newCopy[k] = v
	}
	return docDiff && jsonEqual(oldCopy, newCopy)
}
