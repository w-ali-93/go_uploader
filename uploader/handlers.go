package uploader

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
	"github.com/nfnt/resize"
)

const maxUploadSize = 512 * 1024 // 512 kb
const uploadPath = "./receipts"
const smallestDownScale = 0.1
const largestUpScale = 2.0

func LogRequest(handler http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func UploadFileHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// limit upload size
		r.Body = http.MaxBytesReader(w, r.Body, 2*512*1024)

		// serve webpage for testing
		if r.Method == "GET" {
			t, _ := template.ParseFiles("upload.html")
			t.Execute(w, nil)
			return
		}

		// parse entire multipart form in one go
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			fmt.Printf("Could not parse multipart form: %v\n", err)
			generateError(w, CANT_PARSE_FORM, http.StatusInternalServerError)
			return
		}

		// parse and validate userID from form data
		userID := r.FormValue("userID")
		if len(userID) == 0 {
			generateError(w, MISSING_OR_INVALID_USERID, http.StatusBadRequest)
			return
		}

		// parse and validate file and file header from form data
		file, fileHeader, err := r.FormFile("uploadFile")
		if err != nil {
			generateError(w, INVALID_FILE, http.StatusBadRequest)
			return
		}
		defer file.Close()
		fileSize := fileHeader.Size
		if fileSize > maxUploadSize {
			generateError(w, FILE_TOO_BIG, http.StatusBadRequest)
			return
		}
		fileBytes, err := ioutil.ReadAll(file)
		if err != nil {
			generateError(w, INVALID_FILE, http.StatusBadRequest)
			return
		}

		// parse and validate file type
		detectedFileType := http.DetectContentType(fileBytes)
		switch detectedFileType {
		case "image/jpeg", "image/jpg":
			break
		default:
			generateError(w, INVALID_FILE_TYPE, http.StatusBadRequest)
			return
		}

		// generate folder and image paths
		fileName := uuid.New().String()
		folderPath := filepath.Join(uploadPath, userID)
		imagePath := filepath.Join(folderPath, fileName+".jpg")

		// create folder and write file
		os.MkdirAll(folderPath, 0700)
		newFile, err := os.Create(imagePath)
		if err != nil {
			generateError(w, CANT_WRITE_FILE, http.StatusInternalServerError)
			return
		}
		defer newFile.Close()
		if _, err := newFile.Write(fileBytes); err != nil || newFile.Close() != nil {
			generateError(w, CANT_WRITE_FILE, http.StatusInternalServerError)
			return
		}
		log.Println("UPLOADING:", imagePath)
		w.Write([]byte(fileName))
	})
}

func DownloadFileHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse necessary request parameters
		scale, err := parseScale(r)
		if err != nil {
			generateError(w, err.Error(), http.StatusBadRequest)
			return
		}
		userID, err := parseUserID(r)
		if err != nil {
			generateError(w, err.Error(), http.StatusBadRequest)
			return
		}
		fileName, err := parseFileName(r)
		if err != nil {
			generateError(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Generate and validate image path
		imagePath := filepath.Join(uploadPath, userID, fileName)
		fmt.Println(imagePath)
		if _, err := os.Stat(imagePath); os.IsNotExist(err) {
			generateError(w, FILE_NOT_FOUND_FOR_USER, http.StatusNotFound)
			return
		}

		// Resize image
		resizedImage, err := resizeImage(imagePath, scale)
		if err != nil {
			generateError(w, err.Error(), http.StatusInternalServerError)
		}

		// Respond with resized image
		jpeg.Encode(w, resizedImage, &jpeg.Options{Quality: 95})
	})
}

func parseScale(r *http.Request) (float64, error) {
	scaleRaw, ok := r.URL.Query()["scale"]
	if !ok || len(scaleRaw) == 0 {
		return 0.0, errors.New("MISSING_OR_INVALID_SCALE")
	} else {
		scaleStr := scaleRaw[0]
		scaleVal, err := strconv.ParseFloat(scaleStr, 64)
		if err != nil || (scaleVal < smallestDownScale) || (scaleVal > largestUpScale) {
			return 0.0, errors.New(MISSING_OR_INVALID_SCALE)

		}
		return scaleVal, nil
	}
}

// TODO: Eventually, UserID could be present in the request context as key/value
// pair. This key/value pair could in turn be populated by a middleware that
// performs *actual* authentication e.g. via JWT
func parseUserID(r *http.Request) (string, error) {
	userIDRaw, ok := r.URL.Query()["userid"]
	if !ok || len(userIDRaw) == 0 {
		return "", errors.New(MISSING_OR_INVALID_USERID)
	}
	return userIDRaw[0], nil
}

func parseFileName(r *http.Request) (string, error) {
	fileNameRaw, ok := r.URL.Query()["filename"]
	if !ok || len(fileNameRaw) == 0 {
		return "", errors.New(MISSING_OR_INVALID_FILENAME)
	}
	return fileNameRaw[0], nil
}

func resizeImage(path string, scale float64) (image.Image, error) {
	log.Println("DOWNLOADING:", path)

	imageRawData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.New(CANT_READ_FILE)
	}

	config, err := jpeg.DecodeConfig(bytes.NewReader(imageRawData))
	if err != nil {
		return nil, errors.New(CANT_READ_FILE)
	}
	width := config.Width
	newWidth := uint(float64(width) * scale)

	img, err := jpeg.Decode(bytes.NewReader(imageRawData))
	if err != nil {
		return nil, errors.New(CANT_READ_FILE)
	}

	m := resize.Resize(newWidth, 0, img, resize.NearestNeighbor)
	return m, nil
}
