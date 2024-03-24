# Rivulet

Rivulet is a simple and flexible messaging framework for Go applications, offering seamless integration with different transport mechanisms.

## Overview
Rivulet provides a convenient way for Golang developers to publish and subscribe to messages.

```go
import "github.com/mr-joshcrane/rivulet"
```

![Rivulet](.github/rivulet.png)



Its modular design allows for easy integration of different transports such as in-memory, network-based, and AWS EventBridge. With Rivulet, users can experience a small, stable interface that abstracts the complexities of message handling while providing a seamless communication experience.

## Features

- **Modular Design**: Rivulet offers a modular design, allowing for easy integration of different transport mechanisms.
- **Easy Integration**: The framework is designed to be easy to integrate into existing Go projects, enabling seamless communication using various transport mechanisms.
- **Extensible**: With the ability to define custom transports and message transforms, Rivulet can be easily extended to fit specific use cases.

## Usage

Rivulet provides a simple API for publishing and subscribing to messages. Below is an example of how to use Rivulet to publish messages using AWS EventBridge:

```go
import (
        "context"
        "fmt"
        "github.com/aws/aws-sdk-go-v2/config"
        "github.com/aws/aws-sdk-go-v2/service/eventbridge"
        "github.com/mr-joshcrane/rivulet"
)

func main() {
        cfg, err := config.LoadDefaultConfig(context.Background())
        if err != nil {
                fmt.Println(err)
                return
        }
        eb := eventbridge.NewFromConfig(cfg)
        p := rivulet.NewPublisher(
            "myPublisherId",
            rivulet.WithEventBridgeTransport(eb),
        )
        err = p.Publish("a line")
        if err != nil {
                fmt.Println(err)
        }
}
```



## License

Rivulet is licensed under the MIT License. For more details, see the [LICENSE](LICENSE) file.
