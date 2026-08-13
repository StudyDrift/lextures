package kernel

import "testing"

func TestUnguardedCount_PublicAndNil(t *testing.T) {
	ResetRegistryForTest()
	_ = GET[struct{}](nil, Public(), func(c *Ctx) (struct{}, error) { return struct{}{}, nil }, WithName("p"))
	_ = GET[struct{}](&fakeAccess{}, Guard{}, func(c *Ctx) (struct{}, error) { return struct{}{}, nil }, WithName("n"))
	_ = GET[struct{}](&fakeAccess{}, Authenticated(), func(c *Ctx) (struct{}, error) { return struct{}{}, nil }, WithName("a"))
	if UnguardedCount() != 2 {
		t.Fatalf("unguarded=%d routes=%+v", UnguardedCount(), RegisteredRoutes())
	}
}
