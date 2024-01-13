# clarum-http

The HTTP package of the Clarum framework.

This package contains:

- client and server endpoints
- test actions to send and receive messages
- HTTP specific validation

## Beta warning

The whole framework is currently in beta, so be aware that the setup and API may change.
The documentation below is incomplete. It is just an example to get you started until the complete official documentation will be written.

### Setup
It is recommended to create a separate folder for your integration tests. In this example we will use the name `itests`.

1. Initiate a go project inside the folder you created:
```shell
go mod init <your-package-name>-itests
```

2. Configure dependencies:
```go
require (
	github.com/goclarum/clarum/core v<use-latest-version>
	github.com/goclarum/clarum/http v<use-latest-version>
)
```

3. Now we need to inject the framework into the main thread running the go tests.
Create a file called `setup_test.go` and configure `TestMain` inside of it:
```go
import (
	...
	clarumcore "github.com/goclarum/clarum/core"
)


func TestMain(m *testing.M) {
	clarumcore.Setup()

	result := m.Run()

	clarumcore.Finish()

	os.Exit(result)
}
```

`clarumcore.Setup()` will load the configuration and configure different Clarum internal components.
`clarumcore.Finish()` will make sure that the runtime waits for all test actions to finish.

You can create a `clarum-properties.yaml` file to change different configuration parameters. If such a file does not exist, Clarum will always set defaults.

4. Configure client and/or server endpoints.
```go
import (
...
clarumhttp "github.com/goclarum/clarum/http"
)

var myApiClient = clarumhttp.Http().Client().
    Name("apiClient").
    BaseUrl("http://localhost:8080/myApp").
    Timeout(2000 * time.Millisecond).
    Build()

var thirdPartyServer = clarumhttp.Http().Server().
    Name("thirdPartyServer").
    Port(8083).
    Build()
```

5. Write tests.

The configured endpoints offer you an API to execute test actions. Test actions are how you tell Clarum to do something. 
In this context, the endpoints we have configured above will allow you to send and receive HTTP requests and responses.

All you need to do now is to create a standard go test and use the endpoints to build the logic you want to test.
Remember that you are writing integration tests. This means that before your tests run, your application and all other required infrastructure have to be running.
Clarum offers some functionality to help you with orchestration. See below.

Actions will return errors when executed. These errors will either be caused by something that went wrong when executing the action itself or when a validation has failed.
If you don't want to always check these errors, you can allow Clarum to fail your tests automatically, by passing the current test instance to the action.

Actions can be given the context of the test when executed, by calling `.In(t)`. This will pass the current test instance
to the action which allows it to signal a test failure.


### HTTP Client Endpoint
A client endpoint allows you to send any type of HTTP requests to initiate a use-case in your application.

Here are some examples to get you started. For the complete feature set, check the API.
```go
func TestMyApi(t *testing.T) {

  // send GET request with query parameters 
  myApiClient.In(t).Send("controller", "path").
    Message(message.Get().QueryParam("myParam", "myValue1"))

  // receive response 
  myApiClient.In(t).Receive().
    Json(). // tell Clarum that you expect a JSON payload
    Message(message.Response(http.StatusOK).
      Payload("{" +
        "\"item\": {}" +
      "}")
    )

  // send POST request with JSON 
  myApiClient.In(t).Send().
    Message(message.Put().
      ContentType(constants.ContentTypeJsonHeader).
      Payload("{" +
        "\"name\": \"Bruce Wayne\"" +
      "}")
    )

  // receive response 
  myApiClient.In(t).Receive().
    Json().
    Message(message.Response(http.StatusCreated).
      Payload("{" +
        "\"success\": true," +
        "\"timestamp\": \"@ignore@\"" +
      "}")
    )
}
```


For working examples, check [clarum-samples](https://github.com/go-clarum/samples).
