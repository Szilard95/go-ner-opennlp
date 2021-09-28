package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"io"
	"log"
	"os"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func csvToOnlp(ctoInFileName, ctoOutFileName, ctoInType string) {
	inFile, err := os.Open(ctoInFileName)
	check(err)
	defer inFile.Close()

	outFile, err := os.Create(ctoOutFileName)
	check(err)
	defer outFile.Close()

	r := csv.NewReader(bufio.NewReader(inFile))
	r.Comma = ';'

	if ctoInType == "trainingSet" { // no header in testset csvs
		_, _ = r.Read()
	}

	firstLine := true
	var prevRecord []string = nil

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		chunkType := record[3]
		if ctoInType == "trainingSet" && chunkType == "" { // csv end: ;;;
			break
		}

		if record[0] != "" && !firstLine { // csv: Sentence #
			outFile.WriteString("\n")
		}
		firstLine = false

		if record[0] != "" {
			prevRecord = nil
		}

		if ctoInType == "testSet" {
			outFile.WriteString(record[1] + " ") // just create sentences
		} else {
			outFile.WriteString(iobEncode(prevRecord, record))
		}

		prevRecord = record
	}
}

// IOB encoder:
// BB -> <S> <E><S>...	| close prev B, start
// BI -> <S>...			|
// BO -> <S> <E><S><E>	| close prev B, startend
// IB -> <E> <S>		| close prev I, start
// II -> ...			|
// IO -> <E><S><E>		| close prev I, startend
// OB -> <S><E> <S>		| start
// OI -> ERROR			|
// OO -> <S><E> <S><E>	| startend
func iobEncode(prevRecord, record []string) string {
	encodedChunk := ""
	currType := record[3]
	var prevType string

	if prevRecord != nil {
		prevType = prevRecord[3]
	}

	if currType[0] == 'B' { // begin chunk
		if prevRecord != nil && prevType[0] != 'O' {
			encodedChunk += "<END> "
		}
		encodedChunk += "<START:" + currType[2:] + "> " + record[1] + " "
	} else if currType[0] == 'I' { // inside chunk
		if prevRecord != nil && prevType[0] == 'O' {
			log.Fatal("parsing error")
		}
		encodedChunk += record[1] + " "
	} else { // we outside
		if prevRecord != nil && prevType[0] != 'O' {
			encodedChunk += "<END> "
		}
		encodedChunk += "<START:" + currType + "> " + record[1] + " <END> "
	}

	return encodedChunk
}

func onlpToCsv(otcOnlpFileName, otcTestSetFileName, otcOutFileName string) {
	log.Println("todo")
}

func main() {
	ctoCmd := flag.NewFlagSet("csvToOnlp", flag.ExitOnError)
	ctoInFileName := ctoCmd.String("input", "train.csv", "input csv")
	ctoOutFileName := ctoCmd.String("output", "NERmodel.train", "output openNLP train file")
	ctoInType := ctoCmd.String("inputType", "trainingSet", "input csv type: [trainingSet|testSet]")

	otcCmd := flag.NewFlagSet("onlpToCsv", flag.ExitOnError)
	otcOnlpFileName := otcCmd.String("input", "test.onlp", "input openNLP annotated sentences")
	otcTestSetFileName := otcCmd.String("testSet", "test.csv", "input test set csv")
	otcOutFileName := otcCmd.String("output", "test_pred.csv", "output csv predicted format")

	if len(os.Args) < 2 {
		log.Fatal("expected 'csvToOnlp' or 'onlpToCsv' subcommands")
	}

	switch os.Args[1] {
	case "csvToOnlp":
		ctoCmd.Parse(os.Args[2:])
		log.Println("subcommand 'csvToOnlp'")
		if *ctoInType != "trainingSet" && *ctoInType != "testSet" {
			log.Fatal("expected 'trainingSet' or 'testSet' as 'inputType'")
		}

		csvToOnlp(*ctoInFileName, *ctoOutFileName, *ctoInType)
	case "onlpToCsv":
		otcCmd.Parse(os.Args[2:])
		log.Println("subcommand 'onlpToCsv'")

		onlpToCsv(*otcOnlpFileName, *otcTestSetFileName, *otcOutFileName)
	default:
		log.Fatal("expected 'csvToOnlp' or 'onlpToCsv' subcommands")
	}
}
