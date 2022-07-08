package main

import (
	"errors"
	"io"
	"os"

	"cloud.google.com/go/storage"
	"github.com/pkg/sftp"
	"google.golang.org/api/iterator"
)

var ErrNotImplemented error = errors.New("not implemented")

type Handler struct {
	bucket *storage.BucketHandle
}

type listerat []os.FileInfo

func (f listerat) ListAt(ls []os.FileInfo, offset int64) (int, error) {
	var n int
	if offset >= int64(len(f)) {
		return 0, io.EOF
	}
	n = copy(ls, f[offset:])
	if n < len(ls) {
		return n, io.EOF
	}
	return n, nil
}

func (h *Handler) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	object := h.bucket.Object(r.Filepath[1:])
	reader, err := object.NewReader(r.Context())
	if err != nil {
		return nil, err
	}
	return NewReadAtBuffer(reader)
}

func (h *Handler) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	object := h.bucket.Object(r.Filepath[1:])
	writer := object.NewWriter(r.Context())
	return NewWriteAtBuffer(writer, []byte{}), nil
}

func (h *Handler) Filecmd(r *sftp.Request) error {
	switch r.Method {
	case "Setstat":
		return nil
	case "Rename":
		return ErrNotImplemented
	case "Rmdir", "Remove":
		return ErrNotImplemented
	case "Mkdir":
		object := h.bucket.Object(r.Filepath[1:] + "/")
		writer := object.NewWriter(r.Context())
		err := writer.Close()
		return err
	case "Symlink":
		return ErrNotImplemented
	}
	return nil
}

func (h *Handler) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	switch r.Method {
	case "List":
		prefix := r.Filepath[1:]
		if prefix != "" {
			prefix += "/"
		}

		objects := h.bucket.Objects(r.Context(), &storage.Query{
			Delimiter: "/",
			Prefix:    prefix,
		})

		list := []os.FileInfo{}
		for {
			objAttrs, err := objects.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, err
			}
			if ((prefix != "") && (objAttrs.Prefix == prefix)) || ((objAttrs.Prefix == "") && (objAttrs.Name == prefix)) {
				continue
			}
			list = append(list, &SyntheticFileInfo{
				prefix:  prefix,
				objAttr: objAttrs,
			})
		}
		return listerat(list), nil
	case "Stat":
		if r.Filepath == "/" {
			return listerat([]os.FileInfo{
				&SyntheticFileInfo{
					objAttr: &storage.ObjectAttrs{
						Prefix: "/",
					},
				},
			}), nil
		}
		object := h.bucket.Object(r.Filepath[1:])
		attrs, err := object.Attrs(r.Context())
		if err == storage.ErrObjectNotExist {
			object := h.bucket.Object(r.Filepath[1:] + "/")
			attrs, err = object.Attrs(r.Context())
		}
		if err != nil {
			return nil, err
		}
		file := &SyntheticFileInfo{
			objAttr: attrs,
		}
		return listerat([]os.FileInfo{file}), nil
	case "Readlink":
		return nil, ErrNotImplemented
	}

	return nil, nil
}
