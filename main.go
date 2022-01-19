package main

import (
	"go_uploader/uploader"
	"log"
	"net/http"
)

func main() {
	http.Handle("/upload", uploader.UploadFileHandler())
	http.Handle("/receipts/", uploader.DownloadFileHandler())
	log.Print("Server started on localhost:8080, use /upload for uploading receipts and /receipts?userID=<userID>?receiptID=<receiptID> for downloading")
	log.Fatal(http.ListenAndServe(":8080", uploader.LogRequest(http.DefaultServeMux)))
}
