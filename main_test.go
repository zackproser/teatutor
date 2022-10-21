package main

import "testing"

func TestRenderCorrectColumn(t *testing.T) {
	type TestCase struct {
		A    int
		B    int
		Want string
	}
	testCases := []TestCase{
		{
			A:    0,
			B:    0,
			Want: SuccessEmoji,
		},
		{
			A:    1,
			B:    3,
			Want: FailureEmoji,
		},
		{
			A:    4,
			B:    4,
			Want: SuccessEmoji,
		},
	}
	for _, tc := range testCases {
		got := renderCorrectColumn(tc.A, tc.B)
		if got != tc.Want {
			t.Logf("Wanted %s for inputs: %d and %d but got %s", tc.Want, tc.A, tc.B, got)
			t.Fail()
		}

	}
}

func TestGetHorizontalLineLength(t *testing.T) {
	type TestCase struct {
		Total int
		Want  int
	}

	testCases := []TestCase{
		{
			Total: 30,
			Want:  25,
		},
		{
			Total: 100,
			Want:  85,
		},
	}
	for _, tc := range testCases {
		got := getHorizontalLineLength(tc.Total)
		if got != tc.Want {
			t.Logf("Wanted %d for input %d but got %d", tc.Want, tc.Total, got)
			t.Fail()
		}
	}
}

func makeUserResponseMap() map[int]int {
	m := make(map[int]int)
	m[0] = 1
	m[1] = 2
	m[2] = 1
	m[3] = 0
	m[4] = 1
	m[5] = 2

	return m
}

func testEq(a, b []Answer) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestSortUserResponses(t *testing.T) {
	type TestCase struct {
		Input map[int]int
		Want  []Answer
	}
	testCases := []TestCase{
		{
			Input: makeUserResponseMap(),
			Want: []Answer{
				{QuestionNum: 0, ResponseNum: 1},
				{QuestionNum: 1, ResponseNum: 2},
				{QuestionNum: 2, ResponseNum: 1},
				{QuestionNum: 3, ResponseNum: 0},
				{QuestionNum: 4, ResponseNum: 1},
				{QuestionNum: 5, ResponseNum: 2},
			},
		},
	}
	for _, tc := range testCases {
		got := sortUserResponses(tc.Input)
		if !testEq(tc.Want, got) {
			t.Logf("Wanted %v for input %v but got %v\n", tc.Want, tc.Input, got)
			t.Fail()
		}
	}
}
