package s3fs

import (
	"github.com/minio/minio-go"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type s3fs struct {
	cli *minio.Client
}

func NewS3FS(cli *minio.Client) *s3fs {
	return &s3fs{
		cli: cli,
	}
}

var defaultStat = &syscall.Stat_t{}

// open path
func (s *s3fs) Open(minioPath string) (http.File, error) {

	parent := filepath.Dir(minioPath)
	if parent == minioPath {
		//
		// it's root, show all bucket list
		//
		buckets, err := s.cli.ListBuckets()
		if err != nil {
			return nil, err
		}
		return &s3fsObj{
			tp:        rootType,
			s3fs:      s,
			minioPath: minioPath,
			isDir:     true,
			buckets:   buckets,
			objects:   nil,
			minioObj:  nil,
		}, nil

	} else if strings.Count(minioPath, "/") == 1 {

		//
		// it's bucket, show sub files
		//
		objs := make([]minio.ObjectInfo, 0)

		doneCh := make(chan struct{})
		defer close(doneCh)

		bucketName := strings.Split(minioPath, "/")[1]
		objectCh := s.cli.ListObjectsV2(bucketName, "", false, doneCh)
		for obj := range objectCh {
			if obj.Err != nil {
				return nil, obj.Err
			}
			objs = append(objs, obj)
		}
		return &s3fsObj{
			tp:        bucketType,
			s3fs:      s,
			minioPath: minioPath,
			isDir:     true,
			buckets:   nil,
			objects:   objs,
			minioObj:  nil,
		}, nil

	} else {
		bucketName := strings.Split(minioPath, "/")[1]
		prefix := strings.Join(strings.Split(minioPath, "/")[2:], "/")

		//
		// it's dir, may be is not dir, so we assume it's dir
		//
		objs := make([]minio.ObjectInfo, 0)
		doneCh := make(chan struct{})
		defer close(doneCh)
		objectCh := s.cli.ListObjectsV2(bucketName, prefix+"/", false, doneCh)
		for obj := range objectCh {
			if obj.Err != nil {
				return nil, obj.Err
			}
			objs = append(objs, obj)
		}
		if len(objs) != 0 {
			return &s3fsObj{
				tp:        fileType,
				s3fs:      s,
				minioPath: minioPath,
				isDir:     true,
				buckets:   nil,
				objects:   objs,
				minioObj:  nil,
			}, nil
		}

		//
		// it's file
		//
		obj, err := s.cli.GetObject(bucketName, prefix, minio.GetObjectOptions{})

		if err == nil {
			return &s3fsObj{
				tp:        fileType,
				s3fs:      s,
				minioPath: minioPath,
				isDir:     false,
				buckets:   nil,
				objects:   nil,
				minioObj:  obj,
			}, nil
		}
	}

	return nil, io.EOF
}

type minioType int

const (
	rootType   minioType = 0
	bucketType minioType = 1
	fileType   minioType = 2
)

type s3fsObj struct {
	tp minioType
	*s3fs
	minioPath string
	isDir     bool

	buckets []minio.BucketInfo
	objects []minio.ObjectInfo

	minioObj *minio.Object
}

func (s *s3fsObj) Close() error {
	if !s.isDir {
		return s.minioObj.Close()
	}
	return nil
}

func (s *s3fsObj) Read(p []byte) (int, error) {
	if !s.isDir {
		return s.minioObj.Read(p)
	}
	return 0, nil
}

func (s *s3fsObj) Seek(offset int64, whence int) (int64, error) {
	if !s.isDir {
		return s.minioObj.Seek(offset, whence)
	}
	return 0, nil
}

func (s *s3fsObj) Readdir(_ int) ([]os.FileInfo, error) {
	fileInfos := make([]os.FileInfo, 0)

	if !s.isDir {
		return fileInfos, nil
	}

	switch s.tp {
	case rootType:
		for _, bucket := range s.buckets {
			fileInfos = append(fileInfos, &objectInfo{
				isDir: true,
				name:  bucket.Name,
			})
		}

	case bucketType:
		for _, object := range s.objects {
			var isDir bool

			if strings.HasSuffix(object.Key, "/") {
				isDir = true
			}

			fileInfos = append(fileInfos, &objectInfo{
				isDir:   isDir,
				name:    filepath.Clean(object.Key),
				objInfo: &object,
			})
		}

	case fileType:
		for _, object := range s.objects {

			var isDir bool
			if strings.HasSuffix(object.Key, "/") {
				isDir = true
			}

			fileInfos = append(fileInfos, &objectInfo{
				isDir:   isDir,
				name:    filepath.Base(object.Key),
				objInfo: &object,
			})
		}
	}

	return fileInfos, nil
}

func (s *s3fsObj) Stat() (os.FileInfo, error) {
	if !s.isDir {
		info, err := s.minioObj.Stat()
		if err != nil {
			return nil, os.ErrNotExist
		}
		return &objectInfo{
			objInfo: &info,
			isDir:   false,
			name:    s.minioPath,
		}, nil
	}

	return &objectInfo{
		isDir: true,
		name:  s.minioPath,
	}, nil
}

type objectInfo struct {
	objInfo *minio.ObjectInfo
	isDir   bool
	name    string
}

func (o *objectInfo) Name() string {
	return o.name
}

func (o *objectInfo) Size() int64 {
	if o.isDir {
		return 0
	}
	return o.objInfo.Size
}

func (o *objectInfo) Mode() os.FileMode {
	if o.isDir {
		return os.ModeDir | 0600
	}
	return os.FileMode(0600)
}

func (o *objectInfo) ModTime() time.Time {
	if o.isDir {
		return time.Now()
	}
	return o.objInfo.LastModified
}

func (o *objectInfo) IsDir() bool {
	return o.isDir
}

func (o *objectInfo) Sys() interface{} {
	return defaultStat
}
