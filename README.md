# go-ner-opennlp

Converts custom csv formatted training and test datasets to OpenNLP format, and OpenNLP annotated texts to custom csv.

Csv format (POST can be omitted):
```csv
Sentence #;Word;POS;Tag
Sentence: 1;Test;VB;O
;this;NNP;O
;.;.;O
```

OpenNLP format:
```
<START:named_entitiy_type>Named Entity<END> remaining sentence.
```

Tested OpenNLP version: 1.9.3

## Model 
Convert custom training csv to OpenNLP training set format
```
ner csvToOnlp -input TrainNER.csv -inputType trainingSet -output NERmodel.train
```

Create the model with OpenNLP TokenNameFinderTrainer
```
opennlp TokenNameFinderTrainer -model NERmodel.bin -lang en -data NERmodel.train -encoding UTF-8
```

## Tests
Convert custom csv to OpenNLP test set format
```
ner csvToOnlp -input TestNER.csv -inputType testSet -output NERtest.onlp
```

Run OpenNLP TokenNameFinder with the model and test data
```
cat NERtest.onlp | opennlp TokenNameFinder NERmodel.bin > NERtest_annotated.onlp
```

Merge the custom csv with the annotated OpenNLP test set
```
ner onlpToCsv -input NERtest_annotated.onlp -testSet TestNER.csv -output TestNER_annotated.csv
```