# clarum-http

The HTTP package of the Clarum framework.

This package contains:

- client and server endpoints
- test actions to send and receive messages
- HTTP specific validation

## Beta warning

The whole framework is currently in beta, so be aware that the setup and API may change.
The documentation below is incomplete. It is just an example to get you started until the complete official documentation will be written.

## Setup
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

var productService = clarumhttp.Http().Server().
    Name("productService").
    Port(8083).
    Build()
```

5. Write tests.

The configured endpoints offer you an API to execute test actions. Test actions are how you tell Clarum to do something.
In this context, the endpoints we have configured above will allow you to send and receive HTTP requests and responses.

You will notice that Clarum offers a different way of writing tests. The classical AAA testing pattern, while good for unit testing, is not so good for integration tests.
When writing these kind of tests, use-cases end up being really complex and mocking everything before sending the first request will result in tests that are really hard to read and understand.
This is why Clarum offers a different approach. With test actions you end up writing a test that represents 1:1 the flow of your use-case.

All you need to do now is to create a standard go test and use the endpoints to build the logic you need.
Remember that you are writing integration tests. This means that before your tests run, your application and all other required infrastructure have to be running.
Clarum offers some functionality to help you with orchestration as well. See below.

Actions will return errors when executed. These errors will either be caused by something that went wrong when executing the action itself or when a validation has failed.
If you don't want to always check these errors, you can allow Clarum to fail your tests automatically, by passing the current test instance to the action.

Actions can be given the context of the test when executed, by calling `.In(t)`. This will pass the current test instance
to the action which allows it to signal a test failure.


### HTTP Client Endpoint
A client endpoint allows you to send any type of HTTP request to initiate a use-case in your application.

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

  // send PUT request with JSON 
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

### HTTP Server Endpoint
In a typical scenario, the client will send a request to initiate a use-case, and your service may need to call another service to get some data.
In such a case you will use a server endpoint, which allows you to receive any type of HTTP request sent by the service you are testing and then send a response back.

Here are some examples to get you started. For the complete feature set, check the API.
```go
func TestMyApi(t *testing.T) {

  // receive a GET request from your service
  productService.In(t).Receive().
    Json().
    Message(message.Get("products").
      Payload("{" +
        "\"type\": \"hardware\"" +
      "}")
    )
  // send back an error response
  productService.In(t).Send().
    Message(message.Response(http.StatusInternalServerError))
}

```

### Orchestration
While developing your service, you will probably start it with your IDE in order to debug functionality. You will often run integration tests this way.
But there are also situations when you don't want to have to start your service/infrastructure everytime manually before running the tests.

Clarum offers some actions to automate this as well. The orchestration package gives you the ability to run commands during your tests or test setup.
This way you can either start your service or some infrastructure (docker-compose) required by your tests, before they are executed.

1. Configure your command.
```go
import (
  ...
  "github.com/goclarum/clarum/core/orchestration/command"
)

var myAppInstance = command.Command().
  Components("go", "run", "../main.go").
  Warmup(1 * time.Second).
  Build()
```

1. Setup `start()` & `stop()` in `TestMain`.
```go
func TestMain(m *testing.M) {
  clarumcore.Setup()

  if err := appInstance.Run(); err != nil {
    slog.Error(fmt.Sprintf("Test suite did not start because of startup error - %s", err))
    return
  }

  result := m.Run()

  if err := appInstance.Stop(); err != nil {
    slog.Error(fmt.Sprintf("Test suite ended with shutdown error  - %s", err))
  }
  clarumcore.Finish()

  os.Exit(result)
}
```

**Note**: This kind of setup will certainly change with version 1.0. 
