package uploader

import (
	"log"
	"net/http"
)

const (
	MISSING_OR_INVALID_SCALE    = "MISSING_OR_INVALID_SCALE"
	MISSING_OR_INVALID_USERID   = "MISSING_OR_INVALID_USERID"
	MISSING_OR_INVALID_FILENAME = "MISSING_OR_INVALID_FILENAME"
	FILE_NOT_FOUND_FOR_USER     = "FILE_NOT_FOUND_FOR_USER"
	CANT_PARSE_FORM             = "CANT_PARSE_FORM"
	CANT_READ_FILE              = "CANT_READ_FILE"
	CANT_WRITE_FILE             = "CANT_WRITE_FILE"
	CANT_READ_FILE_TYPE         = "CANT_READ_FILE_TYPE"
	INVALID_FILE_TYPE           = "INVALID_FILE_TYPE"
	INVALID_FILE                = "INVALID_FILE"
	FILE_TOO_BIG                = "FILE_TOO_BIG"
)

func generateError(w http.ResponseWriter, message string, statusCode int) {
	log.Println("ERROR:", message)
	w.WriteHeader(statusCode)
	w.Write([]byte(message))
}
