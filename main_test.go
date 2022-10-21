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
