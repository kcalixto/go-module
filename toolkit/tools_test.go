package toolkit

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
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
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTests {
		pipeReader, pipeWriter := io.Pipe()

		writer := multipart.NewWriter(pipeWriter)

		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			defer writer.Close()
			defer wg.Done()

			imagePath := "testdata/image.png"

			part, err := writer.CreateFormFile("file", imagePath)
			if err != nil {
				t.Error(err)
			}

			file, err := os.Open(imagePath)
			if err != nil {
				t.Error(err)
			}
			defer file.Close()

			img, _, err := image.Decode(file)
			if err != nil {
				t.Error(err)
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}

		}()

		request := httptest.NewRequest("POST", "/", pipeReader)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = e.allowedTypes

		uploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads/", e.renameFile)
		if err != nil && !e.errorExpected {
			t.Error(err)
		}

		if !e.errorExpected {
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); os.IsNotExist(err) {
				t.Errorf("%s: expected file to exists: %s", e.testName, err.Error())
			}

			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName))
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: error expected but not received", e.testName)
		}

		wg.Wait()
	}
}
