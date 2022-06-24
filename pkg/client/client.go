package client

import (
	"container/list"
	"fmt"
	"strings"
	sync2 "sync"

	"github.com/AliyunContainerService/image-syncer/pkg/db/models"
	"github.com/AliyunContainerService/image-syncer/pkg/db/service"
	"github.com/AliyunContainerService/image-syncer/pkg/sync"
	"github.com/AliyunContainerService/image-syncer/pkg/tools"
	"github.com/sirupsen/logrus"
)

type ClientType int

const (
	NormalClient ClientType = iota
	DBClient
)

// Client describes a synchronization client
type Client struct {
	// whether a command based client or db client. 0: command, 1: db.
	clientType ClientType

	// a sync.Task list
	taskList *list.List

	// a URLPair list
	urlPairList *list.List

	// failed list
	failedTaskList         *list.List
	failedTaskGenerateList *list.List

	// success list
	successTaskList *list.List

	config *Config

	routineNum int
	retries    int
	logger     *logrus.Logger

	// mutex
	taskListChan               chan int
	urlPairListChan            chan int
	failedTaskListChan         chan int
	failedTaskGenerateListChan chan int
	successTaskListChan        chan int
}

// URLPair is a pair of source and destination url
type URLPair struct {
	source      string
	destination string
}

func NewSyncClientFromDB(logFile string, routineNum, retries int,
	defaultDestRegistry, defaultDestNamespace string,
	osFilterList, archFilterList []string) (*Client, error) {

	logger := NewFileLogger(logFile)

	config, err := NewSyncConfigFromDB(defaultDestRegistry, defaultDestNamespace, osFilterList, archFilterList)
	if err != nil {
		return nil, fmt.Errorf("generate config error: %v", err)
	}

	return &Client{
		clientType:                 DBClient,
		taskList:                   list.New(),
		urlPairList:                list.New(),
		failedTaskList:             list.New(),
		failedTaskGenerateList:     list.New(),
		successTaskList:            list.New(),
		config:                     config,
		routineNum:                 routineNum,
		retries:                    retries,
		logger:                     logger,
		taskListChan:               make(chan int, 1),
		urlPairListChan:            make(chan int, 1),
		failedTaskListChan:         make(chan int, 1),
		failedTaskGenerateListChan: make(chan int, 1),
		successTaskListChan:        make(chan int, 1),
	}, nil
}

// NewSyncClient creates a synchronization client
func NewSyncClient(configFile, authFile, imageFile, logFile string,
	routineNum, retries int, defaultDestRegistry, defaultDestNamespace string,
	osFilterList, archFilterList []string) (*Client, error) {

	logger := NewFileLogger(logFile)

	config, err := NewSyncConfig(configFile, authFile, imageFile,
		defaultDestRegistry, defaultDestNamespace, osFilterList, archFilterList)
	if err != nil {
		return nil, fmt.Errorf("generate config error: %v", err)
	}

	return &Client{
		clientType:                 NormalClient,
		taskList:                   list.New(),
		urlPairList:                list.New(),
		failedTaskList:             list.New(),
		failedTaskGenerateList:     list.New(),
		successTaskList:            list.New(),
		config:                     config,
		routineNum:                 routineNum,
		retries:                    retries,
		logger:                     logger,
		taskListChan:               make(chan int, 1),
		urlPairListChan:            make(chan int, 1),
		failedTaskListChan:         make(chan int, 1),
		failedTaskGenerateListChan: make(chan int, 1),
	}, nil
}

// Run is main function of a synchronization client
func (c *Client) Run() {
	fmt.Println("Start to generate sync tasks, please wait ...")

	//var finishChan = make(chan struct{}, c.routineNum)

	// open num of goroutines and wait c for close
	openRoutinesGenTaskAndWaitForFinish := func() {
		wg := sync2.WaitGroup{}
		for i := 0; i < c.routineNum; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					urlPair, empty := c.GetAURLPair()
					// no more task to generate
					if empty {
						break
					}
					moreURLPairs, err := c.GenerateSyncTask(urlPair.source, urlPair.destination)
					if err != nil {
						c.logger.Errorf("Generate sync task %s to %s error: %v", urlPair.source, urlPair.destination, err)
						// put to failedTaskGenerateList
						c.PutAFailedURLPair(urlPair)
					}
					if moreURLPairs != nil {
						c.PutURLPairs(moreURLPairs)
					}
				}
			}()
		}
		wg.Wait()
	}

	openRoutinesHandleTaskAndWaitForFinish := func() {
		wg := sync2.WaitGroup{}
		for i := 0; i < c.routineNum; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					task, empty := c.GetATask()
					// no more tasks need to handle
					if empty {
						break
					}
					if err := task.Run(); err != nil {
						// put to failedTaskList
						c.PutAFailedTask(task)
					} else {
						// if success, put it to successTaskList
						c.PutASuccessTask(task)
					}

				}
			}()
		}

		wg.Wait()
	}

	for source, dest := range c.config.GetImageList() {
		c.urlPairList.PushBack(&URLPair{
			source:      source,
			destination: dest,
		})
	}

	// generate sync tasks
	openRoutinesGenTaskAndWaitForFinish()

	fmt.Println("Start to handle sync tasks, please wait ...")

	// generate goroutines to handle sync tasks
	openRoutinesHandleTaskAndWaitForFinish()

	for times := 0; times < c.retries; times++ {
		if c.failedTaskGenerateList.Len() != 0 {
			c.urlPairList.PushBackList(c.failedTaskGenerateList)
			c.failedTaskGenerateList.Init()
			// retry to generate task
			fmt.Println("Start to retry to generate sync tasks, please wait ...")
			openRoutinesGenTaskAndWaitForFinish()
		}

		if c.failedTaskList.Len() != 0 {
			c.taskList.PushBackList(c.failedTaskList)
			c.failedTaskList.Init()
		}

		if c.taskList.Len() != 0 {
			// retry to handle task
			fmt.Println("Start to retry sync tasks, please wait ...")
			openRoutinesHandleTaskAndWaitForFinish()
		}
	}

	fmt.Printf("Finished, %v sync tasks failed, %v tasks generate failed\n", c.failedTaskList.Len(), c.failedTaskGenerateList.Len())
	c.logger.Infof("Finished, %v sync tasks failed, %v tasks generate failed", c.failedTaskList.Len(), c.failedTaskGenerateList.Len())

	// if clientType == DBClient, update sync status of image from false to true
	if c.clientType == DBClient {
		imageService := service.NewImageService()
		for task := c.successTaskList.Front(); task != nil; task = task.Next() {

			task := task.Value.(*sync.Task)

			// TODO: error handle
			source, _ := task.GetSource().GetFullUrl()
			dest, _ := task.GetDestination().GetFullUrl()

			sourceAndTag := strings.Split(source, ":")
			destAndTag := strings.Split(dest, ":")
			if len(sourceAndTag) == 1 {
				sourceUrl := sourceAndTag[0]
				destUrl := destAndTag[0]
				image := &models.Image{Source: sourceUrl, Destination: destUrl}
				imageService.UpdateImageSyncStatus(image, true)
			}
			if len(sourceAndTag) == 2 {
				sourceUrl := sourceAndTag[0]
				tag := sourceAndTag[1]
				destUrl := destAndTag[0]
				image := &models.Image{Source: sourceUrl, Tag: tag, Destination: destUrl}
				imageService.UpdateImageSyncStatus(image, true)
			}
		}
	}

}

// GenerateSyncTask creates synchronization tasks from source and destination url, return URLPair array if there are more than one tags
func (c *Client) GenerateSyncTask(source string, destination string) ([]*URLPair, error) {
	if source == "" {
		return nil, fmt.Errorf("source url should not be empty")
	}

	sourceURL, err := tools.NewRepoURL(source)
	if err != nil {
		return nil, fmt.Errorf("url %s format error: %v", source, err)
	}

	// if dest is not specific, use default registry and namespace
	if destination == "" {
		if c.config.defaultDestRegistry != "" && c.config.defaultDestNamespace != "" {
			destination = c.config.defaultDestRegistry + "/" + c.config.defaultDestNamespace + "/" +
				sourceURL.GetRepoWithTag()
		} else {
			return nil, fmt.Errorf("the default registry and namespace should not be nil if you want to use them")
		}
	}

	destURL, err := tools.NewRepoURL(destination)
	if err != nil {
		return nil, fmt.Errorf("url %s format error: %v", destination, err)
	}

	tags := sourceURL.GetTag()

	// multi-tags config
	if moreTag := strings.Split(tags, ","); len(moreTag) > 1 {
		if destURL.GetTag() != "" && destURL.GetTag() != sourceURL.GetTag() {
			return nil, fmt.Errorf("multi-tags source should not correspond to a destination with tag: %s:%s",
				sourceURL.GetURL(), destURL.GetURL())
		}

		// contains more than one tag
		var urlPairs []*URLPair
		for _, t := range moreTag {
			urlPairs = append(urlPairs, &URLPair{
				source:      sourceURL.GetURLWithoutTag() + ":" + t,
				destination: destURL.GetURLWithoutTag() + ":" + t,
			})
		}

		return urlPairs, nil
	}

	var imageSource *sync.ImageSource
	var imageDestination *sync.ImageDestination

	if auth, exist := c.config.GetAuth(sourceURL.GetRegistry(), sourceURL.GetNamespace()); exist {
		c.logger.Infof("Find auth information for %v, username: %v", sourceURL.GetURL(), auth.Username)
		imageSource, err = sync.NewImageSource(sourceURL.GetRegistry(), sourceURL.GetRepoWithNamespace(), sourceURL.GetTag(),
			auth.Username, auth.Password, auth.Insecure)
		if err != nil {
			return nil, fmt.Errorf("generate %s image source error: %v", sourceURL.GetURL(), err)
		}
	} else {
		c.logger.Infof("Cannot find auth information for %v, pull actions will be anonymous", sourceURL.GetURL())
		imageSource, err = sync.NewImageSource(sourceURL.GetRegistry(), sourceURL.GetRepoWithNamespace(), sourceURL.GetTag(),
			"", "", false)
		if err != nil {
			return nil, fmt.Errorf("generate %s image source error: %v", sourceURL.GetURL(), err)
		}
	}

	// if tag is not specific, return tags
	if sourceURL.GetTag() == "" {
		if destURL.GetTag() != "" {
			return nil, fmt.Errorf("tag should be included both side of the config: %s:%s", sourceURL.GetURL(), destURL.GetURL())
		}

		// get all tags of this source repo
		tags, err := imageSource.GetSourceRepoTags()
		if err != nil {
			return nil, fmt.Errorf("get tags failed from %s error: %v", sourceURL.GetURL(), err)
		}
		c.logger.Infof("Get tags of %s successfully: %v", sourceURL.GetURL(), tags)

		// generate url pairs for tags
		var urlPairs = []*URLPair{}
		for _, tag := range tags {
			urlPairs = append(urlPairs, &URLPair{
				source:      sourceURL.GetURL() + ":" + tag,
				destination: destURL.GetURL() + ":" + tag,
			})
		}
		return urlPairs, nil
	}

	// if source tag is set but without destination tag, use the same tag as source
	destTag := destURL.GetTag()
	if destTag == "" {
		destTag = sourceURL.GetTag()
	}

	if auth, exist := c.config.GetAuth(destURL.GetRegistry(), destURL.GetNamespace()); exist {
		// preprocess for service like aws ecr
		switch tools.RegistryDistinguisher(destURL.GetRegistry()) {
		case tools.AwsRegistry:
			// aws ecr need to create repo before any operation
			c.logger.Infof("Aws ecr auth find in url %v, username: %v", destURL.GetURL(), auth.Username)
			rawAuth, _ := c.config.GetRawAuth(destURL.GetRegistry(), destURL.GetNamespace())
			awsHelper, err := tools.NewAwsHelperFromStaticConfig(rawAuth.Username, rawAuth.Password)
			if err != nil {
				return nil, fmt.Errorf("generate %s image destination error: %v", destURL.GetURL(), err)
			}
			isExist := awsHelper.IsRepositoryExist(destURL.GetRepo())
			if !isExist {
				_, err = awsHelper.CreateRepository(destURL.GetRepo())
				if err != nil {
					return nil, fmt.Errorf("generate %s image destination error: %v", destURL.GetURL(), err)
				}
			}
		}

		c.logger.Infof("Find auth information for %v, username: %v", destURL.GetURL(), auth.Username)
		imageDestination, err = sync.NewImageDestination(destURL.GetRegistry(), destURL.GetRepoWithNamespace(),
			destTag, auth.Username, auth.Password, auth.Insecure)
		if err != nil {
			return nil, fmt.Errorf("generate %s image destination error: %v", sourceURL.GetURL(), err)
		}
	} else {
		c.logger.Infof("Cannot find auth information for %v, push actions will be anonymous", destURL.GetURL())
		imageDestination, err = sync.NewImageDestination(destURL.GetRegistry(), destURL.GetRepoWithNamespace(),
			destTag, "", "", false)
		if err != nil {
			return nil, fmt.Errorf("generate %s image destination error: %v", destURL.GetURL(), err)
		}
	}

	c.PutATask(sync.NewTask(imageSource, imageDestination, c.config.osFilterList, c.config.archFilterList, c.logger))
	c.logger.Infof("Generate a task for %s to %s", sourceURL.GetURL(), destURL.GetURL())
	return nil, nil
}

// GetATask return a sync.Task struct if the task list is not empty
func (c *Client) GetATask() (*sync.Task, bool) {
	c.taskListChan <- 1
	defer func() {
		<-c.taskListChan
	}()

	task := c.taskList.Front()
	if task == nil {
		return nil, true
	}
	c.taskList.Remove(task)

	return task.Value.(*sync.Task), false
}

// PutATask puts a sync.Task struct to task list
func (c *Client) PutATask(task *sync.Task) {
	c.taskListChan <- 1
	defer func() {
		<-c.taskListChan
	}()

	if c.taskList != nil {
		c.taskList.PushBack(task)
	}
}

// GetAURLPair gets a URLPair from urlPairList
func (c *Client) GetAURLPair() (*URLPair, bool) {
	c.urlPairListChan <- 1
	defer func() {
		<-c.urlPairListChan
	}()

	urlPair := c.urlPairList.Front()
	if urlPair == nil {
		return nil, true
	}
	c.urlPairList.Remove(urlPair)

	return urlPair.Value.(*URLPair), false
}

// PutURLPairs puts a URLPair array to urlPairList
func (c *Client) PutURLPairs(urlPairs []*URLPair) {
	c.urlPairListChan <- 1
	defer func() {
		<-c.urlPairListChan
	}()

	if c.urlPairList != nil {
		for _, urlPair := range urlPairs {
			c.urlPairList.PushBack(urlPair)
		}
	}
}

// GetAFailedTask gets a failed task from failedTaskList
func (c *Client) GetAFailedTask() (*sync.Task, bool) {
	c.failedTaskListChan <- 1
	defer func() {
		<-c.failedTaskListChan
	}()

	failedTask := c.failedTaskList.Front()
	if failedTask == nil {
		return nil, true
	}
	c.failedTaskList.Remove(failedTask)

	return failedTask.Value.(*sync.Task), false
}

// PutAFailedTask puts a failed task to failedTaskList
func (c *Client) PutAFailedTask(failedTask *sync.Task) {
	c.failedTaskListChan <- 1
	defer func() {
		<-c.failedTaskListChan
	}()

	if c.failedTaskList != nil {
		c.failedTaskList.PushBack(failedTask)
	}
}

// PutASuccessTask puts a success task to successTaskTaskList
func (c *Client) PutASuccessTask(successTask *sync.Task) {
	c.successTaskListChan <- 1
	defer func() {
		<-c.successTaskListChan
	}()

	if c.successTaskList != nil {
		c.successTaskList.PushBack(successTask)
	}
}

// GetAFailedURLPair get a URLPair from failedTaskGenerateList
func (c *Client) GetAFailedURLPair() (*URLPair, bool) {
	c.failedTaskGenerateListChan <- 1
	defer func() {
		<-c.failedTaskGenerateListChan
	}()

	failedURLPair := c.failedTaskGenerateList.Front()
	if failedURLPair == nil {
		return nil, true
	}
	c.failedTaskGenerateList.Remove(failedURLPair)

	return failedURLPair.Value.(*URLPair), false
}

// PutAFailedURLPair puts a URLPair to failedTaskGenerateList
func (c *Client) PutAFailedURLPair(failedURLPair *URLPair) {
	c.failedTaskGenerateListChan <- 1
	defer func() {
		<-c.failedTaskGenerateListChan
	}()

	if c.failedTaskGenerateList != nil {
		c.failedTaskGenerateList.PushBack(failedURLPair)
	}
}
