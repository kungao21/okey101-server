//go:build solvertest
// +build solvertest

package main

import (
	"encoding/json"
	"fmt"
	"time"
)


func main() {
	hand := []string{
		"B13-1",
		"B11-1",
		"B10-1",
		"B09-2",
		"B09-1",
		"B04-1",
		"B03-2",
		"B03-1",

		"G13-2",
		"G10-2",
		"G09-2",
		"G01-2",

		"K13-1",
		"K11-1",
		"K10-1",
		"K02-1",

		"R13-2",
		"R09-2",
		"R08-2",
		"R06-1",
		"R03-2",

	}

	// "G13-2",
	// "G10-2",
	// "G09-2",
	// "G01-2",

    // "R09-2",
    // "R08-2",
    // "R06-1",
    // "R03-2",

    // "B09-1",
    // "B04-1",
	
    
    // "K11-1",
    // "K10-1",
    // "K02-1"


// 	hand := []string{
// 		"R13-1",
// 		"R10-2",
// 		"R06-1",
// 		"R05-1",
// 		"R01-2",
// 		"K13-2",
// 		"K10-1",
// 		"K04-2",
// 		"K04-1",
// 		"K02-2",
// 		"K01-2",
// 		"JOKER-2",
// 		"G09-1",
// 		"G07-2",
// 		"G06-2",
// 		"G05-2",
// 		"G05-1",
// 		"G04-2",
// 		"G04-1",
// 		"G03-2",
// 		"B13-1",
// 		"R08-1",

// 	}

	indicator := "B02-1"  // test için
	okeyBase  := "B03"    // test için

	// ✅ SADECE RUN (Seri Diz)
	res := SuggestMelds(hand, indicator, okeyBase, SolveRun, 50*time.Millisecond)

	b, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(b))
}
