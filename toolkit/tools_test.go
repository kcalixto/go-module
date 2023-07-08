package toolkit

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

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

	_, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}

}
