package logger

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/sirupsen/logrus"
)

var (
	// ErrCannotCreateIndex Fired if the index is not created
	ErrCannotCreateIndex = fmt.Errorf("cannot create index")
)

// IndexNameFunc get index name
type IndexNameFunc func() string

type fireFunc func(entry *logrus.Entry, hook *ElasticHook) error

type TransType int

const (
	Sync TransType = iota
	Async
	Bulk //TODO
)

// Hook for ElasticSearch
type ElasticHook struct {
	client    *elasticsearch.TypedClient
	host      string
	index     IndexNameFunc
	levels    []logrus.Level
	ctx       context.Context
	ctxCancel context.CancelFunc
	fireFunc  fireFunc
}

type message struct {
	Host      string `json:"Host,omitempty"`
	Timestamp string `json:"@timestamp"`
	File      string `json:"File,omitempty"`
	Func      string `json:"Func,omitempty"`
	Message   string `json:"Message,omitempty"`
	Data      logrus.Fields
	Level     string `json:"Level,omitempty"`
}

type ElasticConfig struct {
	endpoints []string
	cloudId   string
	apiKey    string
}

func NewElasticHook(cfg ElasticConfig, host string, t TransType,
	level logrus.Level, index string) (*ElasticHook, error) {
	c, err := elasticsearch.NewTypedClient(elasticsearch.Config{
		Addresses: cfg.endpoints,
		CloudID:   cfg.cloudId,
		APIKey:    cfg.apiKey,
	})
	if err != nil {
		panic(err)
	}

	indexFunc := func() string {
		return index
	}

	switch t {
	case Sync:
		return newHookFuncAndFireFunc(c, host, level, indexFunc, syncFireFunc)
	case Async:
		return newHookFuncAndFireFunc(c, host, level, indexFunc, asyncFireFunc)
	case Bulk:
		//TODO
		//return newHookFuncAndFireFunc(c, host, level, indexFunc, bulkFireFunc)
		return nil, errors.New("Bulk type is TODO")
	}
	return nil, errors.New("param type is invalid")
}

func newHookFuncAndFireFunc(c *elasticsearch.TypedClient,
	host string, level logrus.Level,
	indexFunc IndexNameFunc, fireFunc fireFunc) (*ElasticHook, error) {

	var levels []logrus.Level
	for _, l := range []logrus.Level{
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
		logrus.TraceLevel,
	} {
		if l <= level {
			levels = append(levels, l)
		}
	}

	ctx, cancel := context.WithCancel(context.TODO())

	// Check if a specified index exists.
	exists, err := c.Indices.Exists(indexFunc()).Do(ctx)
	if err != nil {
		cancel()
		return nil, err
	}
	if !exists {
		createIndex, err := c.Indices.Create(indexFunc()).Do(ctx)
		if err != nil {
			cancel()
			return nil, err
		}
		if !createIndex.Acknowledged {
			cancel()
			return nil, ErrCannotCreateIndex
		}
	}

	return &ElasticHook{
		client:    c,
		host:      host,
		index:     indexFunc,
		levels:    levels,
		ctx:       ctx,
		ctxCancel: cancel,
		fireFunc:  fireFunc,
	}, nil
}

// Fire is required to implement
// Logrus hook
func (hook *ElasticHook) Fire(entry *logrus.Entry) error {
	return hook.fireFunc(entry, hook)
}

// Levels Required for logrus hook implementation
func (hook *ElasticHook) Levels() []logrus.Level {
	return hook.levels
}

// Cancel all calls to elastic
func (hook *ElasticHook) Cancel() {
	hook.ctxCancel()
}

func createMessage(entry *logrus.Entry, hook *ElasticHook) *message {
	level := entry.Level.String()

	if e, ok := entry.Data[logrus.ErrorKey]; ok && e != nil {
		if err, ok := e.(error); ok {
			entry.Data[logrus.ErrorKey] = err.Error()
		}
	}

	var file string
	var function string
	if entry.HasCaller() {
		file = entry.Caller.File
		function = entry.Caller.Function
	}

	return &message{
		hook.host,
		entry.Time.UTC().Format(time.RFC3339Nano),
		file,
		function,
		entry.Message,
		entry.Data,
		strings.ToUpper(level),
	}
}

func syncFireFunc(entry *logrus.Entry, hook *ElasticHook) error {
	_, err := hook.client.
		Index(hook.index()).
		Request(*createMessage(entry, hook)).
		Do(hook.ctx)

	return err
}

func asyncFireFunc(entry *logrus.Entry, hook *ElasticHook) error {
	go syncFireFunc(entry, hook)
	return nil
}

// Create closure with bulk processor tied to fireFunc.
func bulkFireFunc(client *elasticsearch.TypedClient) (fireFunc, error) {
	//TODO
	//processor, err := client.BulkProcessor().
	//	Name("elogrus.v3.bulk.processor").
	//	Workers(2).
	//	FlushInterval(time.Second).
	//	Do(context.Background())

	//return func(entry *logrus.Entry, hook *ElasticHook) error {
	//	r := elastic.NewBulkIndexRequest().
	//		Index(hook.index()).
	//		Type("log").
	//		Doc(*createMessage(entry, hook))
	//	processor.Add(r)
	//	return nil
	//}, err
	return nil, nil
}
