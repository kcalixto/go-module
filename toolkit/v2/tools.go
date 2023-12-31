package toolkit

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

type Tools struct {
	MaxFileSize            int
	AllowedFileTypes       []string
	MaxJSONSize            int
	AllowJSONUnknownFields bool
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

	err := t.CreateDirIfNotExists(uploadDirectory)
	if err != nil {
		return nil, err
	}

	err = r.ParseMultipartForm(int64(t.MaxFileSize))
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

// Create slug from string
// using regex [ˆa-z\d]+
func (t *Tools) Slugify(s string) (slug string, err error) {
	if s == "" {
		return "", errors.New("empty string")
	}

	var re = regexp.MustCompile(`[^a-z\d]+`)

	slug = strings.Trim(
		re.ReplaceAllString(
			strings.ToLower(s), "-",
		), "-",
	)

	if len(slug) == 0 {
		return "", errors.New("slug is zero length")
	}

	return slug, nil
}

// Downloads a file, forcing browser to avoid displaying it in windows using Content-Disposition
func (t *Tools) DownloadStaticFile(w http.ResponseWriter, r *http.Request, pathName, displayName string) {
	// tels browser to download instead of show up
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", displayName))

	http.ServeFile(w, r, pathName)
}

type JSONResponse struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (t *Tools) ReadJSON(w http.ResponseWriter, r *http.Request, data interface{}) (err error) {
	maxBytes := 1024 * 1024 // 1mb
	if t.MaxJSONSize != 0 {
		maxBytes = t.MaxJSONSize
	}

	// this limit is to avoid some DoS attack
	// limiting request payload
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)

	if !t.AllowJSONUnknownFields {
		dec.DisallowUnknownFields()
	}

	err = dec.Decode(data)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSOn type (at character %d)", unmarshalTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field")
			return fmt.Errorf("body contains unknown key %s", fieldName)

		case err.Error() == "http: request body too large":
			return fmt.Errorf("body must not be larger than %d bytes", maxBytes)

		case errors.As(err, &invalidUnmarshalError):
			return fmt.Errorf("error unmarshalling JSON: %s", err.Error())

		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must contain only one json value")
	}

	return nil
}

// WriteJSON Response
func (t *Tools) WriteJSON(w http.ResponseWriter, status int, data interface{}, headers ...http.Header) (err error) {
	out, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if len(headers) > 0 {
		for k, v := range headers[0] {
			w.Header()[k] = v
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_, err = w.Write(out)
	if err != nil {
		return err
	}

	return nil
}

func (t *Tools) ErrorJSONResponse(w http.ResponseWriter, err error, status ...int) error {
	statusCode := http.StatusBadRequest
	if len(status) > 0 {
		statusCode = status[0]
	}

	payload := JSONResponse{
		Error:   true,
		Message: err.Error(),
	}

	return t.WriteJSON(w, statusCode, payload)
}

func (t *Tools) PushJSONToRemote(uri string, data interface{}, client ...*http.Client) (*http.Response, int, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, 0, err
	}

	httpClient := &http.Client{}
	if len(client) > 0 {
		httpClient = client[0]
	}

	request, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, 0, err
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()

	return response, response.StatusCode, nil
}
