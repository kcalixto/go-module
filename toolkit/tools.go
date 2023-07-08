package toolkit

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

type Tools struct {
	MaxFileSize      int
	AllowedFileTypes []string
}

func (t *Tools) RandomString(length int) string {
	// rune is alias for int32
	s := make([]rune, length)
	// this code is converting these strings to their ASCII codes
	// since we're not trespassing 127, we're safe to use byte or int
	// but if we'd had emojis or other special characters, rune would've made
	// all the difference
	r := []rune(randomStringSource)
	// output: [97 98 99 100 101 102 103 104 105 106 107 108 109 110 111... etc]

	for i := range s {
		// now crazy shit starts, get ready...
		// This function generates a prime number containing 2ˆlen(r-1)+1 to 2ˆlen(r)-1 bits
		// considering our 64 characters long it's like:
		// 9.2 quintillions to 18 quintillions bits
		// remember that it generates a prime number?
		// to validate it, the code performs a primality test
		// one very known and used one is Miller-Rabin test
		// that iterates the generated number like 20 times
		// before returning the number for us
		// All of that to generated a 99.9% random number
		p, _ := rand.Prime(rand.Reader, len(r))

		// this part is just converting it to int64
		// so we can work with it properly :D
		x := p.Uint64()

		// this one is doing the same
		// converting so we can work better
		y := uint64(len(r))

		// And here we're taking the rest of
		// the division of 9.who_fucking_cares quintillions by 64
		// And this number can never be greater than 63
		// due to Modular Arithmetic:
		// 100 = 1 x 64 + 36
		// 200 = 3 x 64 + 8
		// 300 = 4 x 64 + 52
		// And it's not limited by 64
		// any positive number(x) / any positive number(y)
		// can never have a rest greater than dividend(y) - 1
		s[i] = r[x%y]
	}

	// finally we're converting all that rune array(bits of characters) into string
	return string(s)
}

type UploadFile struct {
	NewFileName      string
	OriginalFileName string
	FileSize         int64
}

func (t *Tools) UploadOneFile(r *http.Request, uploadDir string, rename ...bool) (*UploadFile, error) {
	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}

	files, err := t.UploadFiles(r, uploadDir, renameFile)
	if err != nil {
		return nil, err
	}

	return files[0], nil
}

func (t *Tools) UploadFiles(r *http.Request, uploadDirectory string, rename ...bool) ([]*UploadFile, error) {
	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}

	var uploadedFiles []*UploadFile

	if t.MaxFileSize == 0 {
		t.MaxFileSize = 1024 * 1024 * 1024 // ~1gb
	}

	err := r.ParseMultipartForm(int64(t.MaxFileSize))
	if err != nil {
		return nil, errors.New("uploaded file is too big")
	}

	for _, fileHeaders := range r.MultipartForm.File {
		for _, hdr := range fileHeaders {
			uploadedFiles, err = func(uploadedFiles []*UploadFile) ([]*UploadFile, error) {
				var uploadedFile UploadFile
				infile, err := hdr.Open()
				if err != nil {
					return nil, err
				}
				defer infile.Close()

				// detect file type
				detectFileType := func(file multipart.File) (string, error) {
					// most of files just need the first 512 bytes
					// to identify their type
					buff := make([]byte, 512)
					_, err = file.Read(buff)
					if err != nil {
						return "", err
					}

					// we could've passed the entire file size
					// here, but any way this function will only
					// take the first 512 bytes

					return http.DetectContentType(buff), nil
				}

				allowed := false
				fileType, err := detectFileType(infile)

				if len(t.AllowedFileTypes) > 0 {
					for _, x := range t.AllowedFileTypes {
						if strings.EqualFold(fileType, x) {
							allowed = true
							break
						}
					}
				} else {
					allowed = true
				}

				if !allowed {
					return nil, errors.New("uploaded file type not allowed")
				}

				// Since we've read the first 512 bytes of the file
				// it's necessary to reset the file pointer
				// so it won't be broken file or something
				_, err = infile.Seek(0, 0)
				if err != nil {
					return nil, err
				}

				if renameFile {
					uploadedFile.NewFileName = fmt.Sprintf("%s%s", t.RandomString(10), filepath.Ext(hdr.Filename))
				} else {
					uploadedFile.NewFileName = hdr.Filename
				}

				uploadedFile.OriginalFileName = hdr.Filename

				var outfile *os.File
				defer outfile.Close()

				if outfile, err = os.Create(filepath.Join(uploadDirectory, uploadedFile.NewFileName)); err != nil {
					return nil, err
				} else {
					fileSize, err := io.Copy(outfile, infile)
					if err != nil {
						return nil, err
					}

					uploadedFile.FileSize = fileSize
				}

				uploadedFiles = append(uploadedFiles, &uploadedFile)

				return uploadedFiles, nil
			}(uploadedFiles)
			if err != nil {
				return uploadedFiles, err
			}
		}
	}

	return uploadedFiles, nil
}

// Creates a directory if not exists
// and all necessary parents
func (t *Tools) CreateDirIfNotExists(path string) (err error) {
	// permission
	const mode = 0755

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, mode)
		if err != nil {
			return err
		}
	}

	return nil
}
