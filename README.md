# minio-s3fs-go

### demo

![demos](https://raw.githubusercontent.com/qingsong-he/minio-s3fs-go/master/demos.png)

### Usage

##### Install minio from docker

```
docker pull minio/minio
docker run -p 9000:9000 --name minio1 -e "MINIO_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE" -e "MINIO_SECRET_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"  -d minio/minio server /data
```

##### test code

```
func TestNewS3FS(t *testing.T) {
	cli, err := minio.New(
		"localhost:9000",
		"AKIAIOSFODNN7EXAMPLE",
		"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		false,
	)
	if err != nil {
		t.Fatal(err)
	}

	http.HandleFunc(
		"/static/",
		http.StripPrefix("/static/", http.FileServer(NewS3FS(cli))).ServeHTTP,
	)
	err = http.ListenAndServe(":3000", nil)
	if err != nil {
		t.Fatal(err)
	}
}
```

##### bucket and key
 
Take 'http://localhost:3000/static/bucket2/sub1/tmp1.txt' as an example. 'bucket2' is the name of the bucket, and 'sub1/tmp1.txt' is the key of the file in the bucket.

Access 'http://localhost:3000/static/' get all the buckets

### Implementation Details

Many implementations is inspired by https://github.com/harshavardhana/s3www, thanks for harshavardhana's good work!

