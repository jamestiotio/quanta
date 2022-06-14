package sink

// S3Sink - Support for SELECT * INTO "s3://..."

import (
	"context"
	"database/sql/driver"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	u "github.com/araddon/gou"
	"github.com/araddon/qlbridge/exec"
	"github.com/araddon/qlbridge/plan"
	"github.com/araddon/qlbridge/value"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/rlmcpherson/s3gof3r"
	pgs3 "github.com/xitongsys/parquet-go-source/s3v2"
	"github.com/xitongsys/parquet-go/parquet"
	"github.com/xitongsys/parquet-go/source"
	"github.com/xitongsys/parquet-go/writer"
)

type (
	// S3CSVSink - State for AWS S3 implemention of Sink interface for CSV output.
	S3CSVSink struct {
		outBucket      *s3gof3r.Bucket
		outBucketConf  *s3gof3r.Config
		writer         io.WriteCloser
		csvWriter      *csv.Writer
		headersWritten bool
		delimiter      rune
		assumeRoleArn  string
		acl            string
		sseKmsKeyId    string
		config         *aws.Config
	}
)

type (
	// S3ParquetSink - State for AWS S3 implemention of Sink interface for Parquet output.
	S3ParquetSink struct {
		csvWriter      *writer.CSVWriter
		outFile        source.ParquetFile
		md        	   []string
		assumeRoleArn  string
		acl            string
		sseKmsKeyId    string		
		config         *aws.Config
	}
)

var (
	// Ensure that we implement the Sink interface
	// to ensure this can be called form the Into task
	_ exec.Sink = (*S3CSVSink)(nil)
	_ exec.Sink = (*S3ParquetSink)(nil)
)

// NewS3Sink - Construct S3Sink
func NewS3Sink(ctx *plan.Context, path string, params map[string]interface{}) (exec.Sink, error) {

	var s exec.Sink
	s = &S3CSVSink{}
	if fmt, ok := params["format"]; ok && fmt == "parquet" {
		s = &S3ParquetSink{}
		u.Debug("Format == Parquet")
	}
	err := s.Open(ctx, path, params)
	if err != nil {
		u.Errorf("Error creating S3 sink '%v' for path '%v'\n", err, path)
	}
	return s, err
}

// Open CSV session to S3
func (s *S3CSVSink) Open(ctx *plan.Context, bucketpath string, params map[string]interface{}) error {

	if delimiter, ok := params["delimiter"]; !ok {
		s.delimiter = '\t'
	} else {
		ra := []rune(delimiter.(string))
		s.delimiter = ra[0]
	}

	if assumeRoleArn, ok := params["assumeRoleArn"]; ok {
		s.assumeRoleArn = assumeRoleArn.(string)
		u.Debug("assumeRoleArn : '%s'\n", s.assumeRoleArn)
	}
	
	if acl, ok := params["acl"]; ok {
		s.acl = acl.(string)
		u.Debug("ACL : '%s'\n", s.acl)
	}

	if sseKmsKeyId, ok := params["sseKmsKeyId"]; ok {
		s.sseKmsKeyId = sseKmsKeyId.(string)
		u.Debug("kms : '%s'\n", s.sseKmsKeyId)
	}

	bucket, file, err := parseBucketName(bucketpath)
	if err != nil {
		return err
	}

	// k, err := s3gof3r.EnvKeys() // get S3 keys from environment
	k, err := s3gof3r.InstanceKeys() // get S3 keys from environment
	if err != nil {
		return err
	}
	s3 := s3gof3r.New("", k)
	s.outBucket = s3.Bucket(bucket)
	s.outBucketConf = s3gof3r.DefaultConfig
	s.outBucketConf.Concurrency = 16
	w, err := s.outBucket.PutWriter(file, nil, s.outBucketConf)
	if err != nil {
		return err
	}
	s.writer = w
	s.csvWriter = csv.NewWriter(w)
	s.csvWriter.Comma = s.delimiter
	return nil
}

// Next batch of output data
func (s *S3CSVSink) Next(dest []driver.Value, colIndex map[string]int) error {
	if !s.headersWritten {
		cNames := make([]string, len(colIndex))
		for k, i := range colIndex {
			cNames[i] = k
		}
		headers := []byte(strings.Join(cNames, string(s.delimiter)) + "\n")
		if s.writer == nil {
			return fmt.Errorf("nil writer, open call must have failed")
		}
		if _, err := s.writer.Write(headers); err != nil {
			return err
		}
		s.headersWritten = true
	}
	vals := make([]string, len(dest))
	for i, v := range dest {
		if val, ok := v.(string); ok {
			vals[i] = strings.TrimSpace(val)
		} else if val, ok := v.(value.StringValue); ok {
			vals[i] = strings.TrimSpace(val.Val())
		} else if val, ok := v.(value.BoolValue); ok {
			vals[i] = strings.TrimSpace(val.ToString())
		} else {
			vals[i] = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	}
	if err := s.csvWriter.Write(vals); err != nil {
		return err
	}
	return nil
}

// Close S3 session.
func (s *S3CSVSink) Close() error {
	// Channel closed so close the output chunk
	if s.writer == nil {
		return nil
	}
	if s.csvWriter != nil {
		s.csvWriter.Flush()
	}
	if err := s.writer.Close(); err != nil {
		return err
	}
	return nil
}

// Open Parquet session to S3
func (s *S3ParquetSink) Open(ctx *plan.Context, bucketpath string, params map[string]interface{}) error {

	bucket, file, err := parseBucketName(bucketpath)
	if err != nil {
		return err
	}

	u.Infof("Parquet Sink: Bucket Path for parquet write: %s", bucketpath)
	u.Infof("Parquet Sink: Bucket for parquet write: %s", bucket)
	u.Infof("Parquet Sink: File for parquet write: %s", file)

	region := "us-east-1"
	if r, ok := params["region"]; ok {
		region = r.(string)
	}

	if assumeRoleArn, ok := params["assumeRoleArn"]; ok {
		s.assumeRoleArn = assumeRoleArn.(string)
		u.Infof("Parquet Sink: Assuming Arn Role : ", s.assumeRoleArn)
	}

	if acl, ok := params["acl"]; ok {
		s.acl = acl.(string)
		u.Infof("Parquet Sink: ACL : ", s.acl)
	}

	if sseKmsKeyId, ok := params["sseKmsKeyId"]; ok {
		s.sseKmsKeyId = sseKmsKeyId.(string)
		u.Infof("Parquet Sink: sseKmsKeyId : ", s.sseKmsKeyId)
	}

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		u.Errorf("Parquet Sink: Could not load the default config: %v",err)
	}

	var s3svc *s3.Client

	if s.assumeRoleArn != "" {		
		client := sts.NewFromConfig(cfg)
		provider := stscreds.NewAssumeRoleProvider(client, s.assumeRoleArn)			
		value,err := provider.Retrieve(context.TODO())

		if err != nil {
			return fmt.Errorf("Failed to retrieve credentials: %v",err)
		}

		u.Debugf("Credential values: %v", value)
		u.Debugf("Access Key: ", value.AccessKeyID)
		u.Debugf("Secret Key: ", value.SecretAccessKey)
		u.Debugf("Session Token: ", value.SessionToken)

		cfg.Credentials = awsv2.NewCredentialsCache(provider)
		_,err = cfg.Credentials.Retrieve(context.TODO())
		if err != nil {
			return fmt.Errorf("Failed to retrieve credentials from cache: %v", err)
		}	

		s3svc = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.Region = region
			o.Credentials = provider
			o.RetryMaxAttempts = 10
		})
	} else {
		s3svc = s3.NewFromConfig(cfg, 	func(o *s3.Options) {
			o.Region = region
			o.RetryMaxAttempts = 10
		})
	}

	if s3svc == nil {
		return fmt.Errorf("Failed creating S3 session.")
	}

	// Create S3 service client
	u.Infof("Parquet Sink: Opening Output S3 path s3://%s/%s", bucket, file)
	s.outFile, err = pgs3.NewS3FileWriterWithClient(context.Background(), s3svc, bucket, file, nil, func(p *s3.PutObjectInput){
		p.SSEKMSKeyId = aws.String(s.sseKmsKeyId)
		p.ServerSideEncryption = "aws:kms"
		p.ACL = types.ObjectCannedACL(s.acl)
	})

	if err != nil {
		u.Error(err)
		return err
	}

	// Construct parquet metadata
	s.md = make([]string, len(ctx.Projection.Proj.Columns))
	for i, v := range ctx.Projection.Proj.Columns {
		switch v.Type {
		case value.IntType:
			s.md[i] = fmt.Sprintf("name=%s, type=INT64", v.As)
		case value.NumberType:
			s.md[i] = fmt.Sprintf("name=%s, type=FLOAT", v.As)
		case value.BoolType:
			s.md[i] = fmt.Sprintf("name=%s, type=BOOLEAN", v.As)
		default:
			s.md[i] = fmt.Sprintf("name=%s, type=UTF8, encoding=PLAIN_DICTIONARY", v.As)
		}
	}

	s.csvWriter, err = writer.NewCSVWriter(s.md, s.outFile, 4)
	if err != nil {
		u.Errorf("Parquet Sink: Can't create csv writer %s", err)
		return err
	}

	s.csvWriter.RowGroupSize = 128 * 1024 * 1024 //128M
	s.csvWriter.CompressionType = parquet.CompressionCodec_SNAPPY
	return nil
}

// Next batch of output data
func (s *S3ParquetSink) Next(dest []driver.Value, colIndex map[string]int) error {

	vals := make([]string, len(dest))
	for i, v := range dest {
		if val, ok := v.(string); ok {
			vals[i] = strings.TrimSpace(val)
		} else if val, ok := v.(value.StringValue); ok {
			vals[i] = strings.TrimSpace(val.Val())
		} else if val, ok := v.(value.BoolValue); ok {
			vals[i] = strings.TrimSpace(val.ToString())
		} else {
			vals[i] = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	}

	rec := make([]*string, len(vals))
	for j := 0; j < len(vals); j++ {
		rec[j] = &vals[j]
	}
	if err := s.csvWriter.WriteString(rec); err != nil {
		return err
	}

	return nil
}

// Close S3 session.
func (s *S3ParquetSink) Close() error {

	if err := s.csvWriter.WriteStop(); err != nil {
		return fmt.Errorf("Parquet Sink: WriteStop error %v", err)
	}
	if err := s.outFile.Close(); err != nil {
		u.Errorf("Parquet Sink: Outfile close error: %v", err)
	}
	u.Infof("Parquest file successfully written.")
	return nil
}

func parseBucketName(bucketPath string) (bucket string, file string, err error) {

	noScheme := strings.Replace(strings.ToLower(bucketPath), "s3://", "", 1)
	split_path := strings.SplitN(noScheme, "/",2)
	bucket = split_path[0]
	file = split_path[1]

	if bucket == "" {
		err = fmt.Errorf("no bucket specified")
		return
	}
	if file == "" {
		err = fmt.Errorf("no file specified")
		return
	}

	return
}
