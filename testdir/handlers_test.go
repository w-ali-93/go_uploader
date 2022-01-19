package test

import (
	"bytes"
	"go_uploader/uploader"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const testArtifactsDir = "./testartifacts"
const testUserID = "a1bc"
const testExistingReceiptFilename = "2b72ad28-9ce9-4e3e-b9f9-2af537ae0d4f"
const testReceiptsDir = "./receipts"
const testReceiptsOfUserDir = testReceiptsDir + "/" + testUserID

func setupSuite(t testing.T) func(t testing.T) {
	os.MkdirAll(testReceiptsOfUserDir, 0700)
	uploader.Copy("./testartifacts/valid_image.jpg", testReceiptsOfUserDir+"/"+testExistingReceiptFilename+".jpg")

	return func(t testing.T) {
		os.RemoveAll("./receipts")
		log.Println("Tearing down test suite")
	}
}

func TestUploadFile(t *testing.T) {
	teardownSuite := setupSuite(*t)
	defer teardownSuite(*t)

	table := []struct {
		testName             string
		inputFileName        string
		expectedResponseCode int
	}{
		{"Valid Image", "valid_image.jpg", http.StatusOK},
		{"Too Large Image Size", "too_large_image.jpg", http.StatusBadRequest},
	}

	for _, tc := range table {
		t.Run(tc.testName, func(t *testing.T) {
			UploadFileTester(t, tc.inputFileName, tc.expectedResponseCode)
		})
	}
}

func UploadFileTester(t *testing.T, inputFileName string, expectedResponseCode int) {
	//Set up a pipe to avoid buffering
	pr, pw := io.Pipe()
	//writer to transform input it to multipart form data
	//and write it to io.Pipe
	writer := multipart.NewWriter(pw)

	go func() {
		defer writer.Close()
		//write testUserID as userID form value
		writer.WriteField("userID", testUserID)

		//create the form data field 'fileupload'
		part, err := writer.CreateFormFile("uploadFile", inputFileName)
		if err != nil {
			t.Error(err)
		}

		//load test image
		imgRawData, err := ioutil.ReadFile(testArtifactsDir + "/" + inputFileName)
		if err != nil {
			t.Error(err)
		}
		img, err := jpeg.Decode(bytes.NewReader(imgRawData))
		if err != nil {
			t.Error(err)
		}
		//pass test image to Encode which writes
		//it to the multipart writer as field "uploadFile"
		err = jpeg.Encode(part, img, &jpeg.Options{Quality: 95})
		if err != nil {
			t.Error(err)
		}
	}()

	//Read from the pipe which receives data
	//from the multipart writer, which, in turn,
	//receives data from jpeg.Encode().
	request := httptest.NewRequest("POST", "/", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	// Record http response from request
	response := httptest.NewRecorder()
	handler := uploader.UploadFileHandler()
	handler.ServeHTTP(response, request)

	// Test response
	if expectedResponseCode != response.Code {
		t.Errorf("Expected %d, received %d", expectedResponseCode, response.Code)
		return
	}

	// Test that valid JPEG exists, if response code is 200
	if response.Code == 200 {
		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
			log.Fatal(err)
		}
		savedFileNameWithouExtension := string(bodyBytes)
		intendedReceiptDestination := testReceiptsDir + "/" + testUserID + "/" + savedFileNameWithouExtension + ".jpg"
		if _, err := os.Stat(intendedReceiptDestination); os.IsNotExist(err) {
			t.Error("Expected file " + intendedReceiptDestination + " to exist")
		}
	}
}

func TestDownloadFile(t *testing.T) {
	teardownSuite := setupSuite(*t)
	defer teardownSuite(*t)

	table := []struct {
		testName             string
		inputFileName        string
		scale                string
		userID               string
		expectedResponseCode int
	}{
		{"Valid Request", testExistingReceiptFilename + ".jpg", "1.5", testUserID, http.StatusOK},
		{"Invalid Request Invalid Scale", testExistingReceiptFilename + ".jpg", "-1.0", testUserID, http.StatusBadRequest},
		{"Invalid Request Invalid UserID", testExistingReceiptFilename + ".jpg", "1.0", "INVALID_ID", http.StatusNotFound},
		{"Invalid Request Invalid FileName", "INVALID_FILE_NAME" + ".jpg", "1.0", testUserID, http.StatusNotFound},
	}

	for _, tc := range table {
		t.Run(tc.testName, func(t *testing.T) {
			DownloadFileTester(t, tc.scale, tc.userID, tc.inputFileName, tc.expectedResponseCode)
		})
	}
}

func DownloadFileTester(t *testing.T, scale string, userID string, inputFileName string, expectedResponseCode int) {
	request := httptest.NewRequest("GET", "/", nil)

	q := request.URL.Query()
	q.Add("scale", scale)
	q.Add("userid", userID)
	q.Add("filename", inputFileName)
	request.URL.RawQuery = q.Encode()

	// Record http response from request
	response := httptest.NewRecorder()
	handler := uploader.DownloadFileHandler()
	handler.ServeHTTP(response, request)

	// Test response
	if expectedResponseCode != response.Code {
		t.Errorf("Expected %d, received %d", expectedResponseCode, response.Code)
		return
	}

	// Test that valid JPEG is returned, if response code is 200
	if response.Code == 200 {
		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
			log.Fatal(err)
		}
		_, err = jpeg.Decode(bytes.NewReader(bodyBytes))
		if err != nil {
			t.Error(err)
		}
	}
}
