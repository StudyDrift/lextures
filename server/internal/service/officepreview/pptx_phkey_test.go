package officepreview

import "testing"

func TestPhKeyIsCoveredMatchesByIdx(t *testing.T) {
	t.Parallel()
	covered := map[phKey]bool{
		{typ: "", idx: "1"}: true,
	}
	if !phKeyIsCovered(covered, phKey{typ: "body", idx: "1"}) {
		t.Fatal("body idx=1 should be covered when slide defines idx=1 without type")
	}
}

func TestLookupPhInfoMatchesByIdx(t *testing.T) {
	t.Parallel()
	phMap := map[phKey]phInfo{
		{typ: "", idx: "1"}: {cx: 100, cy: 200},
	}
	info, ok := lookupPhInfo(phMap, phKey{typ: "body", idx: "1"})
	if !ok || info.cx != 100 || info.cy != 200 {
		t.Fatalf("lookupPhInfo() = (%v, %v)", info, ok)
	}
}
