package crawler

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Source struct {
	parsed *url.URL
}

func (s *s3Source) Crawl(ctx context.Context, opts CrawlOptions) (*CrawlResult, error) {
	bucket, prefix, region, endpoint := s3ParseURL(s.parsed)
	client, err := s3Connect(ctx, region, endpoint, opts)
	if err != nil {
		return nil, err
	}
	return crawlS3(ctx, client, bucket, prefix, opts)
}

func s3ParseURL(u *url.URL) (bucket, prefix, region, endpoint string) {
	bucket = u.Hostname()
	prefix = strings.TrimPrefix(u.Path, "/")
	region = u.Query().Get("region")
	if region == "" {
		region = "us-east-1"
	}
	endpoint = u.Query().Get("endpoint")
	return
}

func s3Connect(ctx context.Context, region, endpoint string, opts CrawlOptions) (*s3.Client, error) {
	var cfgOpts []func(*config.LoadOptions) error
	cfgOpts = append(cfgOpts, config.WithRegion(region))

	if opts.AuthUser != "" && opts.AuthPass != "" {
		cfgOpts = append(cfgOpts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(opts.AuthUser, opts.AuthPass, ""),
		))
	}

	cfg, err := config.LoadDefaultConfig(ctx, cfgOpts...)
	if err != nil {
		return nil, fmt.Errorf("S3 config failed: %w", err)
	}

	var s3Opts []func(*s3.Options)
	if endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		})
	}

	return s3.NewFromConfig(cfg, s3Opts...), nil
}

func crawlS3(ctx context.Context, client *s3.Client, bucket, prefix string, opts CrawlOptions) (*CrawlResult, error) {
	files, wordSet, err := s3ListAll(ctx, client, bucket, prefix)
	if err != nil {
		return nil, err
	}

	downloader := func(f discoveredFile) ([]byte, error) {
		return s3Download(ctx, client, bucket, f.path)
	}
	pageContexts, secrets, processed := processFiles("S3", files, wordSet, opts, downloader)

	return buildFileResult("s3", bucket+"/"+prefix, wordSet, pageContexts, secrets, processed, opts), nil
}

func s3ListAll(ctx context.Context, client *s3.Client, bucket, prefix string) ([]discoveredFile, map[string]struct{}, error) {
	wordSet := make(map[string]struct{})
	var files []discoveredFile

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	paginator := s3.NewListObjectsV2Paginator(client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("S3 list failed: %w", err)
		}
		for _, obj := range page.Contents {
			key := aws.ToString(obj.Key)
			if strings.HasSuffix(key, "/") {
				continue
			}
			if obj.Size != nil && *obj.Size > 50*1024*1024 {
				continue
			}
			name := path.Base(key)
			addNamesToWordSet(name, wordSet)
			files = append(files, discoveredFile{path: key, name: name})
		}
	}

	return files, wordSet, nil
}

func s3Download(ctx context.Context, client *s3.Client, bucket, key string) ([]byte, error) {
	resp, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	return io.ReadAll(resp.Body)
}
