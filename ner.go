package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"io"
	"log"
	"os"
	"strings"
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

		if ctoInType == "trainingSet" && record[1] == "" { // csv end: ;;;
			break
		}

		if record[0] != "" && !firstLine { // csv: Sentence #
			outFile.WriteString("\n")
		}
		firstLine = false

		if record[0] != "" {
			prevRecord = nil
		}

		if ctoInType == "trainingSet" {
			outFile.WriteString(iobEncode(prevRecord, record)) // this encoding might not be needed if OpenNLP can handle IOB prefixed data, in that case just use this:
			// outFile.WriteString("<START:" + record[3] + "> " + record[1] + " <END> ")
		} else {
			outFile.WriteString(record[1] + " ") // just create sentences
		}

		prevRecord = record
	}
}

// IOB to OpenNLP format encoder:
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
	typeField := 3
	if len(record) <= 3 { // in case the POS field is missing from the training set
		typeField = 2
	}
	currType := record[typeField]
	var prevType string

	if prevRecord != nil {
		prevType = prevRecord[typeField]
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

type prediction struct {
	word       string
	annotation string
}

func onlpTestSetWorker(onlpFile *os.File, predictions chan<- prediction) {
	scanner := bufio.NewScanner(onlpFile)

	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), " ")
		onlpDecode(parts, predictions)
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

// OpenNLP format to IOB decoder
func onlpDecode(parts []string, predictions chan<- prediction) {
	pred := prediction{}
	for _, t := range parts {
		if strings.HasPrefix(t, "<START:") {
			pred.annotation = t[len("<START:") : len(t)-1]
			if pred.annotation != "O" {
				pred.annotation = "B-" + pred.annotation
			}
		} else if t != "<END>" {
			pred.word = t
			predictions <- pred
			if pred.annotation[0] == 'B' {
				pred.annotation = "I" + pred.annotation[1:]
			}
		}
	}
}

func onlpToCsv(otcOnlpFileName, otcTestSetFileName, otcOutFileName string) {
	testSetFile, err := os.Open(otcTestSetFileName)
	check(err)
	defer testSetFile.Close()

	onlpFile, err := os.Open(otcOnlpFileName)
	check(err)
	defer onlpFile.Close()

	outFile, err := os.Create(otcOutFileName)
	check(err)
	defer outFile.Close()

	testSetReader := csv.NewReader(bufio.NewReader(testSetFile))
	testSetReader.Comma = ';'

	outFileWriter := csv.NewWriter(outFile)
	outFileWriter.Comma = ';'
	defer outFileWriter.Flush()

	headerWritten := false
	predictions := make(chan prediction)

	go onlpTestSetWorker(onlpFile, predictions) // processes onlp file

	for {
		record, err := testSetReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if !headerWritten {
			if len(record) > 2 {
				outFileWriter.Write([]string{"Sentences", "Word", "POST", "Predicted"})
			} else {
				outFileWriter.Write([]string{"Sentences", "Word", "Predicted"})
			}
			headerWritten = true
		}

		pred := <-predictions
		if record[1] != pred.word {
			log.Fatal("Out-of-sync while processing the onlp and csv files: ", record[1], " != ", pred.word)
		}

		outFileWriter.Write(append(record, pred.annotation))
	}
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
