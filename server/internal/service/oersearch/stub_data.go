package oersearch

import "strings"

// stubCatalog returns deterministic sample results for dev, tests, and offline mode.
func stubCatalog(provider string) []Result {
	switch provider {
	case "oer_commons":
		return []Result{
			{
				ID: "oc-photosynthesis-1", Title: "Photosynthesis for High School Biology",
				Description: "Open lesson on light reactions and the Calvin cycle with diagrams.",
				URL: "https://www.oercommons.org/courses/photosynthesis-hs-bio",
				PreviewURL: "https://www.oercommons.org/courses/photosynthesis-hs-bio",
				Provider: "oer_commons", LicenseSPDX: "CC-BY-4.0", LicenseLabel: "CC BY",
				GradeLevel: "9-12", Subject: "Biology",
				Attribution: "Photosynthesis for High School Biology © Author, licensed CC BY 4.0",
			},
			{
				ID: "oc-algebra-1", Title: "Introduction to Linear Equations",
				Description: "Algebra unit covering slope-intercept form and graphing lines.",
				URL: "https://www.oercommons.org/courses/linear-equations-intro",
				PreviewURL: "https://www.oercommons.org/courses/linear-equations-intro",
				Provider: "oer_commons", LicenseSPDX: "CC-BY-4.0", LicenseLabel: "CC BY",
				GradeLevel: "9-12", Subject: "Mathematics",
				Attribution: "Introduction to Linear Equations © Author, licensed CC BY 4.0",
			},
			{
				ID: "oc-ml-1", Title: "Machine Learning Foundations (Open Course)",
				Description: "Survey of supervised learning, evaluation, and ethical use of ML.",
				URL: "https://www.oercommons.org/courses/ml-foundations-open",
				PreviewURL: "https://www.oercommons.org/courses/ml-foundations-open",
				Provider: "oer_commons", LicenseSPDX: "CC-BY-4.0", LicenseLabel: "CC BY",
				GradeLevel: "Higher Education", Subject: "Computer Science",
				Attribution: "Machine Learning Foundations © Author, licensed CC BY 4.0",
			},
			{
				ID: "oc-chem-nc", Title: "Organic Chemistry Lab Manual (NC)",
				Description: "Lab procedures for introductory organic chemistry.",
				URL: "https://www.oercommons.org/courses/org-chem-lab-nc",
				Provider: "oer_commons", LicenseSPDX: "CC-BY-NC-4.0", LicenseLabel: "CC BY-NC",
				GradeLevel: "Higher Education", Subject: "Chemistry",
			},
			{
				ID: "oc-history-sa", Title: "World History Reader (ShareAlike)",
				Description: "Primary source collection with SA remix terms.",
				URL: "https://www.oercommons.org/courses/world-history-sa",
				Provider: "oer_commons", LicenseSPDX: "CC-BY-SA-4.0", LicenseLabel: "CC BY-SA",
				GradeLevel: "9-12", Subject: "History",
			},
			{
				ID: "oc-essay-nd", Title: "Essay Writing Guide (No Derivatives)",
				Description: "Writing handbook with ND restrictions.",
				URL: "https://www.oercommons.org/courses/essay-guide-nd",
				Provider: "oer_commons", LicenseSPDX: "CC-BY-ND-4.0", LicenseLabel: "CC BY-ND",
				GradeLevel: "9-12", Subject: "English",
			},
		}
	case "merlot":
		return []Result{
			{
				ID: "merlot-algebra", Title: "College Algebra Problem Sets",
				Description: "Practice sets for polynomials, factoring, and rational expressions.",
				URL: "https://www.merlot.org/merlot/viewMaterial.htm?id=algebra-college",
				PreviewURL: "https://www.merlot.org/merlot/viewMaterial.htm?id=algebra-college",
				Provider: "merlot", LicenseSPDX: "CC-BY-4.0", LicenseLabel: "CC BY",
				GradeLevel: "Higher Education", Subject: "Mathematics",
				Attribution: "College Algebra Problem Sets © Contributor, CC BY 4.0",
			},
			{
				ID: "merlot-photosynthesis", Title: "Photosynthesis Interactive Simulation",
				Description: "Higher-ed simulation of chloroplast electron transport.",
				URL: "https://www.merlot.org/merlot/viewMaterial.htm?id=photosynthesis-sim",
				Provider: "merlot", LicenseSPDX: "CC-BY-4.0", LicenseLabel: "CC BY",
				GradeLevel: "Higher Education", Subject: "Biology",
			},
		}
	case "openstax":
		return []Result{
			{
				ID: "os-bio-ch6", Title: "OpenStax Biology: Photosynthesis",
				Description: "Chapter on photosynthetic reactions and energy capture in plants.",
				URL: "https://openstax.org/books/biology-2e/pages/8-2-the-light-dependent-reactions-of-photosynthesis",
				PreviewURL: "https://openstax.org/books/biology-2e/pages/8-2-the-light-dependent-reactions-of-photosynthesis",
				Provider: "openstax", LicenseSPDX: "CC-BY-4.0", LicenseLabel: "CC BY",
				GradeLevel: "Higher Education", Subject: "Biology",
				Attribution: "OpenStax Biology 2e © Rice University, licensed CC BY 4.0",
			},
			{
				ID: "os-algebra-ch1", Title: "OpenStax Elementary Algebra: Real Numbers",
				Description: "Chapter introducing algebraic expressions and linear equations.",
				URL: "https://openstax.org/books/elementary-algebra-2e/pages/1-introduction",
				PreviewURL: "https://openstax.org/books/elementary-algebra-2e/pages/1-introduction",
				Provider: "openstax", LicenseSPDX: "CC-BY-4.0", LicenseLabel: "CC BY",
				GradeLevel: "9-12", Subject: "Mathematics",
				Attribution: "OpenStax Elementary Algebra 2e © Rice University, licensed CC BY 4.0",
			},
		}
	default:
		return nil
	}
}

func filterStubResults(all []Result, params SearchParams) []Result {
	q := strings.ToLower(strings.TrimSpace(params.Query))
	var out []Result
	for _, r := range all {
		if params.License != "" && !MatchesLicenseFilter(r.LicenseSPDX, params.License) {
			continue
		}
		if params.Subject != "" && !strings.Contains(strings.ToLower(r.Subject), strings.ToLower(params.Subject)) {
			continue
		}
		if params.Level != "" && r.GradeLevel != "" &&
			!strings.Contains(strings.ToLower(r.GradeLevel), strings.ToLower(params.Level)) {
			continue
		}
		if q != "" {
			hay := strings.ToLower(r.Title + " " + r.Description + " " + r.Subject)
			if !strings.Contains(hay, q) {
				continue
			}
		}
		out = append(out, r)
	}
	return out
}
