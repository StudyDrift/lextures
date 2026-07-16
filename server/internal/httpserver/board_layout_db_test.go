package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBoardLayout_SwitchToColumnsCreatesUnsorted_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, _ := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	boardID := createBoardViaAPI(t, h, tok, cc)

	rr2 := httptest.NewRecorder()
	postBody, _ := json.Marshal(map[string]any{"contentType": "text", "body": "hello"})
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts", bytes.NewReader(postBody))
	req2.Header.Set("Authorization", "Bearer "+tok)
	req2.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusCreated {
		t.Fatalf("create post: %d %s", rr2.Code, rr2.Body.String())
	}
	var post map[string]any
	_ = json.Unmarshal(rr2.Body.Bytes(), &post)
	postID := post["id"].(string)

	rr3 := httptest.NewRecorder()
	patch, _ := json.Marshal(map[string]any{"layout": "columns"})
	req3 := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+cc+"/boards/"+boardID, bytes.NewReader(patch))
	req3.Header.Set("Authorization", "Bearer "+tok)
	req3.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusOK {
		t.Fatalf("patch layout: %d %s", rr3.Code, rr3.Body.String())
	}
	var updatedBoard map[string]any
	_ = json.Unmarshal(rr3.Body.Bytes(), &updatedBoard)
	if updatedBoard["layout"] != "columns" {
		t.Fatalf("layout want columns, got %v", updatedBoard["layout"])
	}

	rr4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID+"/sections", nil)
	req4.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr4, req4)
	if rr4.Code != http.StatusOK {
		t.Fatalf("list sections: %d %s", rr4.Code, rr4.Body.String())
	}
	var secResp struct {
		Sections []map[string]any `json:"sections"`
	}
	_ = json.Unmarshal(rr4.Body.Bytes(), &secResp)
	unsortedID := ""
	for _, s := range secResp.Sections {
		if s["title"] == "Unsorted" {
			unsortedID = s["id"].(string)
		}
	}
	if unsortedID == "" {
		t.Fatalf("Unsorted missing: %+v", secResp.Sections)
	}

	rr5 := httptest.NewRecorder()
	req5 := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID, nil)
	req5.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr5, req5)
	var updatedPost map[string]any
	_ = json.Unmarshal(rr5.Body.Bytes(), &updatedPost)
	if updatedPost["sectionId"] != unsortedID {
		t.Fatalf("post should land in Unsorted, got %v", updatedPost["sectionId"])
	}
}

func TestBoardLayout_ArrangeAuthzAndLock_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, teacherTok, cc, courseID := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()
	_, studentTok := enrollSecondUser(t, ctx, pool, courseID, cc, "student")

	boardID := createBoardViaAPI(t, h, teacherTok, cc)

	// VC.6: canArrange defaults false; enable for author-arrange coverage (layout lock still applies).
	rrPolicy := httptest.NewRecorder()
	policy, _ := json.Marshal(map[string]any{"canArrange": true})
	reqPolicy := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+cc+"/boards/"+boardID, bytes.NewReader(policy))
	reqPolicy.Header.Set("Authorization", "Bearer "+teacherTok)
	reqPolicy.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rrPolicy, reqPolicy)
	if rrPolicy.Code != http.StatusOK {
		t.Fatalf("enable canArrange: %d %s", rrPolicy.Code, rrPolicy.Body.String())
	}

	rr2 := httptest.NewRecorder()
	postBody, _ := json.Marshal(map[string]any{"contentType": "text", "body": "student note"})
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts", bytes.NewReader(postBody))
	req2.Header.Set("Authorization", "Bearer "+studentTok)
	req2.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusCreated {
		t.Fatalf("student create post: %d %s", rr2.Code, rr2.Body.String())
	}
	var post map[string]any
	_ = json.Unmarshal(rr2.Body.Bytes(), &post)
	postID := post["id"].(string)

	rr3 := httptest.NewRecorder()
	arr, _ := json.Marshal(map[string]any{
		"sortIndex": 1.5,
		"position":  map[string]any{"x": 10, "y": 20, "w": 240, "h": 160},
	})
	req3 := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID+"/arrange", bytes.NewReader(arr))
	req3.Header.Set("Authorization", "Bearer "+studentTok)
	req3.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusOK {
		t.Fatalf("student arrange own: %d %s", rr3.Code, rr3.Body.String())
	}

	rr4 := httptest.NewRecorder()
	lock, _ := json.Marshal(map[string]any{"layoutLocked": true})
	req4 := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+cc+"/boards/"+boardID, bytes.NewReader(lock))
	req4.Header.Set("Authorization", "Bearer "+teacherTok)
	req4.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr4, req4)
	if rr4.Code != http.StatusOK {
		t.Fatalf("lock: %d %s", rr4.Code, rr4.Body.String())
	}

	rr5 := httptest.NewRecorder()
	arr2, _ := json.Marshal(map[string]any{"sortIndex": 2})
	req5 := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID+"/arrange", bytes.NewReader(arr2))
	req5.Header.Set("Authorization", "Bearer "+studentTok)
	req5.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr5, req5)
	if rr5.Code != http.StatusForbidden {
		t.Fatalf("locked arrange want 403, got %d %s", rr5.Code, rr5.Body.String())
	}

	rr6 := httptest.NewRecorder()
	req6 := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID+"/arrange", bytes.NewReader(arr2))
	req6.Header.Set("Authorization", "Bearer "+teacherTok)
	req6.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr6, req6)
	if rr6.Code != http.StatusOK {
		t.Fatalf("teacher arrange locked: %d %s", rr6.Code, rr6.Body.String())
	}
}

func TestBoardLayout_SectionDeleteMovesToUnsorted_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, _ := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	boardID := createBoardViaAPI(t, h, tok, cc)

	rrL := httptest.NewRecorder()
	patch, _ := json.Marshal(map[string]any{"layout": "columns"})
	reqL := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+cc+"/boards/"+boardID, bytes.NewReader(patch))
	reqL.Header.Set("Authorization", "Bearer "+tok)
	reqL.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rrL, reqL)

	rrS := httptest.NewRecorder()
	secBody, _ := json.Marshal(map[string]any{"title": "Pros"})
	reqS := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/sections", bytes.NewReader(secBody))
	reqS.Header.Set("Authorization", "Bearer "+tok)
	reqS.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rrS, reqS)
	if rrS.Code != http.StatusCreated {
		t.Fatalf("create section: %d %s", rrS.Code, rrS.Body.String())
	}
	var sec map[string]any
	_ = json.Unmarshal(rrS.Body.Bytes(), &sec)
	secID := sec["id"].(string)

	rrP := httptest.NewRecorder()
	postBody, _ := json.Marshal(map[string]any{"contentType": "text", "body": "idea"})
	reqP := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts", bytes.NewReader(postBody))
	reqP.Header.Set("Authorization", "Bearer "+tok)
	reqP.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rrP, reqP)
	var post map[string]any
	_ = json.Unmarshal(rrP.Body.Bytes(), &post)
	postID := post["id"].(string)

	rrA := httptest.NewRecorder()
	arr, _ := json.Marshal(map[string]any{"sectionId": secID, "sortIndex": 0})
	reqA := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID+"/arrange", bytes.NewReader(arr))
	reqA.Header.Set("Authorization", "Bearer "+tok)
	reqA.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rrA, reqA)
	if rrA.Code != http.StatusOK {
		t.Fatalf("arrange into Pros: %d %s", rrA.Code, rrA.Body.String())
	}

	rrD := httptest.NewRecorder()
	reqD := httptest.NewRequest(http.MethodDelete, "/api/v1/courses/"+cc+"/boards/"+boardID+"/sections/"+secID, nil)
	reqD.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rrD, reqD)
	if rrD.Code != http.StatusNoContent {
		t.Fatalf("delete section: %d %s", rrD.Code, rrD.Body.String())
	}

	rrG := httptest.NewRecorder()
	reqG := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID, nil)
	reqG.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rrG, reqG)
	var moved map[string]any
	_ = json.Unmarshal(rrG.Body.Bytes(), &moved)
	if moved["sectionId"] == nil || moved["sectionId"] == secID {
		t.Fatalf("post should move to Unsorted, got %v", moved["sectionId"])
	}
}

func TestBoardLayout_InvalidCoords_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, _ := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	boardID := createBoardViaAPI(t, h, tok, cc)

	rrP := httptest.NewRecorder()
	postBody, _ := json.Marshal(map[string]any{"contentType": "text", "body": "pin"})
	reqP := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts", bytes.NewReader(postBody))
	reqP.Header.Set("Authorization", "Bearer "+tok)
	reqP.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rrP, reqP)
	var post map[string]any
	_ = json.Unmarshal(rrP.Body.Bytes(), &post)
	postID := post["id"].(string)

	rrA := httptest.NewRecorder()
	arr, _ := json.Marshal(map[string]any{"lat": 120.0, "lng": 0.0})
	reqA := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID+"/arrange", bytes.NewReader(arr))
	reqA.Header.Set("Authorization", "Bearer "+tok)
	reqA.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rrA, reqA)
	if rrA.Code != http.StatusBadRequest {
		t.Fatalf("invalid lat want 400, got %d %s", rrA.Code, rrA.Body.String())
	}
}
