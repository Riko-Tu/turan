package controllers

import (
	"testing"
	"time"
)

/*
url中key的生成与解析单元测试
*/

func TestCreateParseUrl(t *testing.T) {
	var expectedShareId int64
	expectedShareId = 100
	time1 := time.Now().Add(60 * time.Second).Unix()
	time2 := time.Now().Truncate(60 * time.Second).Unix()
	// 正确key
	testCase1, err := CreateKey(expectedShareId, time1)
	// 过期key
	testCase2, err := CreateKey(expectedShareId, time2)
	// 无效key
	testCase3 := "abcdefghijklmnopqrstuvwxyz123456789"
	if err != nil {
		t.Error(err.Error())
	}

	// case1 正确key
	shareId, exp, err := ParseKey(testCase1)
	if err != nil {
		t.Error(err.Error())
	}
	if shareId != expectedShareId || exp != false {
		t.Error("expected correct latexId, non-expired")
	}

	// case2 过期key
	shareId, exp, err = ParseKey(testCase2)
	if err != nil {
		t.Error(err.Error())
	}
	if shareId != expectedShareId || exp != true {
		t.Error("expected correct latexId, expired")
	}

	// case3 无效key
	if _, _, err = ParseKey(testCase3); err == nil {
		t.Error("expected error from incorrect key")
	}

}
