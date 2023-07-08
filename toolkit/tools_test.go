package toolkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

type RoundTripFunc func(request *http.Request) *http.Response

func (exec RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return exec(req), nil
}

func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestTools_PushJSONToRemote(t *testing.T) {
	client := NewTestClient(func(request *http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("ok")),
			Header:     make(http.Header),
		}
	})

	var testTools Tools
	var foo struct {
		Bar string `json:"bar"`
	}

	foo.Bar = "bar"

	_, _, err := testTools.PushJSONToRemote("http://example.com/some/path", foo, client)
	if err != nil {
		t.Error("failed to call remote url", err)
	}
}

func TestTools_RandomString(t *testing.T) {
	var testTools Tools

	s := testTools.RandomString(10)
	if len(s) != 10 {
		t.Error("Wrong random string length returned")
	}
}

var uploadTests = []struct {
	testName      string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{testName: "allowed no rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: false, errorExpected: false},
	{testName: "allowed rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: true, errorExpected: false},
	{testName: "not allowed file type", allowedTypes: []string{"image/jpeg"}, renameFile: false, errorExpected: true},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTests {
		// pipes are a connection between two processes
		// using the same buffer, when we try to read from
		// some pipe that's empty, the process wait until there's
		// something. If it tries to write, but it's already full
		// it waits until there's space to write.
		pipeReader, pipeWriter := io.Pipe()

		// multipart is a common way to handle big files
		// without over-using available memory or network
		writer := multipart.NewWriter(pipeWriter)

		// wait groups are wait groups, c'mon
		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			// We need to close the writer, otherwise the reader
			// will wait forever, it's kinda of a sign
			// that we've finished the writing
			defer writer.Close()
			defer wg.Done()

			// absolute path for our test image
			imagePath := "testdata/image.png"

			// this creates a new form-data header
			// with the given field name and file name
			// so os.Open can read the file
			part, err := writer.CreateFormFile("file", imagePath)
			if err != nil {
				t.Error(err)
			}

			// Open our image
			file, err := os.Open(imagePath)
			if err != nil {
				t.Error(err)
			}
			// we need to close the reader to avoid
			// over-usage of resources
			defer file.Close()

			// Decode our image to wite it into the pipe later
			img, _, err := image.Decode(file)
			if err != nil {
				t.Error(err)
			}

			// "save" our image into the pipewriter
			// to do the http request encoded in .png
			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}

		}()

		// Since the pipeReader blocks read if the buffer
		// is empty, this request is always ready to execute
		// but awaiting pipewriter to input data to do the http request
		request := httptest.NewRequest("POST", "/", pipeReader)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = e.allowedTypes

		uploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads/", e.renameFile)
		if err != nil && !e.errorExpected {
			t.Error(err)
		}

		if !e.errorExpected {
			// os.Stat returns some information about the file in question
			// if it does not exists we get an error return, since this only
			// runs after our writing, it's a great way to check if our
			// testing function worked properly
			//
			// os.IsNotExist validate that it is a "file not found"
			// type of error
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); os.IsNotExist(err) {
				t.Errorf("%s: expected file to exists: %s", e.testName, err.Error())
			}

			// clean up deleting uploaded files
			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName))
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: error expected but not received", e.testName)
		}

		wg.Wait()
	}
}

func TestTools_UploadOneFile(t *testing.T) {
	// pipes are a connection between two processes
	// using the same buffer, when we try to read from
	// some pipe that's empty, the process wait until there's
	// something. If it tries to write, but it's already full
	// it waits until there's space to write.
	pipeReader, pipeWriter := io.Pipe()

	// multipart is a common way to handle big files
	// without over-using available memory or network
	writer := multipart.NewWriter(pipeWriter)

	// wait groups are wait groups, c'mon

	go func() {
		// We need to close the writer, otherwise the reader
		// will wait forever, it's kinda of a sign
		// that we've finished the writing
		defer writer.Close()

		// absolute path for our test image
		imagePath := "testdata/image.png"

		// this creates a new form-data header
		// with the given field name and file name
		// so os.Open can read the file
		part, err := writer.CreateFormFile("file", imagePath)
		if err != nil {
			t.Error(err)
		}

		// Open our image
		file, err := os.Open(imagePath)
		if err != nil {
			t.Error(err)
		}
		// we need to close the reader to avoid
		// over-usage of resources
		defer file.Close()

		// Decode our image to wite it into the pipe later
		img, _, err := image.Decode(file)
		if err != nil {
			t.Error(err)
		}

		// "save" our image into the pipewriter
		// to do the http request encoded in .png
		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}

	}()

	// Since the pipeReader blocks read if the buffer
	// is empty, this request is always ready to execute
	// but awaiting pipewriter to input data to do the http request
	request := httptest.NewRequest("POST", "/", pipeReader)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	var testTools Tools

	uploadedFile, err := testTools.UploadOneFile(request, "./testdata/uploads/", true)
	if err != nil {
		t.Error(err)
	}

	// os.Stat returns some information about the file in question
	// if it does not exists we get an error return, since this only
	// runs after our writing, it's a great way to check if our
	// testing function worked properly
	//
	// os.IsNotExist validate that it is a "file not found"
	// type of error
	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFile.NewFileName)); os.IsNotExist(err) {
		t.Errorf("expected file to exists: %s", err.Error())
	}

	// clean up deleting uploaded files
	_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFile.NewFileName))
}

func TestTools_CreateDirIfNotExists(t *testing.T) {
	var testTool Tools

	testDir := "./test-data/myDir"

	err := testTool.CreateDirIfNotExists(testDir)
	if err != nil {
		t.Error(err)
	}

	err = os.Remove(testDir)
	if err != nil {
		fmt.Println("failed to clean up test space: ", err.Error())
	}
}

var slugTests = []struct {
	testName      string
	s             string
	expected      string
	errorExpected bool
}{
	{
		testName:      "valid string",
		s:             "Hello My Dear!!",
		expected:      "hello-my-dear",
		errorExpected: false,
	},
	{
		testName:      "empty string",
		s:             "",
		expected:      "",
		errorExpected: true,
	},
	{
		testName:      "complex string",
		s:             "Now is the time for all GOOD men! - fish & such &^123",
		expected:      "now-is-the-time-for-all-good-men-fish-such-123",
		errorExpected: false,
	},
	{
		testName:      "japanese string",
		s:             "こんにちは、ベイビー",
		expected:      "",
		errorExpected: true,
	},
	{
		testName:      "japanese string and roman characters",
		s:             "Hello Baby! こんにちは、ベイビー",
		expected:      "hello-baby",
		errorExpected: false,
	},
}

func TestTools_Slugify(t *testing.T) {
	var testTools Tools

	for _, e := range slugTests {
		slug, err := testTools.Slugify(e.s)
		if err != nil {
			if !e.errorExpected {
				t.Errorf("%s: error received when not expected %s", e.testName, err.Error())
			}
		}

		if !e.errorExpected {
			if slug != e.expected {
				t.Errorf("%s: wrong slug returned, expected %s but got %s", e.testName, e.expected, slug)
			}
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	// we're using ResponseRecorder to get
	// info about response after
	// since we cannot use a pointer to request
	// to save the information
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	var testTool Tools

	newFileName := "netflix.jpeg"
	testTool.DownloadStaticFile(rr, req, "./testdata", "image.jpeg", newFileName)

	res := rr.Result()
	defer res.Body.Close()

	imageBytesLength := "24423"
	if res.Header["Content-Length"][0] != imageBytesLength {
		t.Error("wrong content length of", res.Header["Content-Length"][0])
	}

	if res.Header["Content-Disposition"][0] != fmt.Sprintf("attachment; filename=\"%s\"", newFileName) {
		t.Error("wrong content disposition")
	}

	_, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}

}

var jsonTests = []struct {
	testName      string
	json          string
	errorExpected bool
	maxSize       int
	allowUnknown  bool
}{
	{
		testName:      "good json",
		json:          `{"foo": "bar"}`,
		errorExpected: false,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		testName:      "badly formatted json",
		json:          `{"foo":}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		testName:      "incorrect type",
		json:          `{"foo": 1}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		testName:      "two json files",
		json:          `{"foo": "bar"}{"foo": "bar"}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		testName:      "empty body",
		json:          "",
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		testName:      "syntax error in json",
		json:          `{"foo": bar"}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		testName:      "unknown field in json",
		json:          `{"bar": "foo"}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		testName:      "allow unknown fields in json",
		json:          `{"bar": "foo"}`,
		errorExpected: false,
		maxSize:       1024,
		allowUnknown:  true,
	},
	{
		testName:      "missing field name",
		json:          `{aloha: "foo"}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		testName:      "file too large",
		json:          `{"foo": "bar"}`,
		errorExpected: true,
		maxSize:       1,
		allowUnknown:  false,
	},
	{
		testName:      "not json",
		json:          "aloha",
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
}

func TestTools_ReadJSON(t *testing.T) {
	var testTool Tools

	for _, e := range jsonTests {
		testTool.MaxJSONSize = e.maxSize
		testTool.AllowJSONUnknownFields = e.allowUnknown

		var decodedJSON struct {
			Foo string `json:"foo"`
		}

		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(e.json)))
		if err != nil {
			t.Error(err)
		}

		rr := httptest.NewRecorder()

		err = testTool.ReadJSON(rr, req, &decodedJSON)
		if e.errorExpected {
			if err == nil {
				t.Errorf("%s: error expected but non recieved", e.testName)
			}
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: error not expected but got one: %s", e.testName, err.Error())
		}

		req.Body.Close()
	}
}

func TestTools_WriteJSON(t *testing.T) {
	var testTools Tools

	rr := httptest.NewRecorder()

	payload := JSONResponse{
		Error:   false,
		Message: "foo",
	}

	headers := make(http.Header)
	headers.Add("FOO", "BAR")

	err := testTools.WriteJSON(rr, http.StatusOK, payload)
	if err != nil {
		t.Errorf("failed to write json: %v", err)
	}

}

func TestTools_ErrorJSON(t *testing.T) {
	var testTools Tools

	rr := httptest.NewRecorder()

	someerror := errors.New("text string")
	err := testTools.ErrorJSONResponse(rr, someerror, http.StatusServiceUnavailable)
	if err != nil {
		t.Error(err)
	}

	var payload JSONResponse

	decoder := json.NewDecoder(rr.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		t.Error("received an error decoding JSON", err)
	}

	if !payload.Error {
		t.Error("error set to false in JSON, and it should be true")
	}

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("wrong status code returned, expected StatusUnavailable, but got %d", rr.Code)
	}
}
