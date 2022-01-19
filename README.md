**Go uploader program**  
This program runs a file upload/download server on localhost:8080.
Use */upload* for uploading receipts and
*/receipts?scale=\<scale\>&userid=\<userID\>&filename=\<filename\>* for downloading.

**Starting the program**  
```go run main.go```

**Testing the source code**  
```cd ./testdir && go test``` 