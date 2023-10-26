# Kinesis Package

The `inskinesis` package is a Go library designed to facilitate streaming data to Amazon Kinesis streams. This README provides an overview of the package's functionality and usage.

## Table of Contents

- [Introduction](#introduction)
- [Installation](#installation)
- [Getting Started](#getting-started)
- [Package Structure](#package-structure)
- [Usage](#usage)
- [Error Handling](#error-handling)
- [Contributing](#contributing)
- [License](#license)

## Introduction

The `inskinesis` package is designed to make it easier to stream data to Amazon Kinesis streams in your Go applications. It provides a simple interface for sending records to a Kinesis stream while handling batching, partitioning, and retries. This can be especially useful for applications that generate a high volume of data and need to send it to Kinesis efficiently.

## Installation

To use the `inskinesis` package in your Go project, you can install it using Go modules. Run the following command in your project directory:

```bash
go get github.com/go-pkg/inskinesis
```

## Getting Started

Here's a quick guide on how to get started with the `inskinesis` package:

1. Import the package in your Go code:

   ```go
   import "github.com/go-pkg/inskinesis"
   ```

2. Create a configuration for your Kinesis stream:

   ```go
   config := inskinesis.Config{
       Region:                 "your-aws-region",
       StreamName:             "your-kinesis-stream-name",
       Partitioner:            nil, // Optionally provide a partitioner function
       MaxStreamBatchSize:     100, // Maximum size of each batch of records
       MaxStreamBatchByteSize: 1024 * 1024, // Maximum size in bytes for each batch
       MaxBatchSize:           500, // Maximum size of the log buffer
       MaxGroup:               10, // Maximum number of concurrent groups for sending records
   }
   ```

3. Create a Kinesis stream instance:

   ```go
   stream, err := inskinesis.NewKinesis(config)
   if err != nil {
       // Handle the error
   }
   ```

4. Send records to the Kinesis stream:

   ```go
   // Send a single record
   stream.Put(yourRecord)

   // Send multiple records
   records := []interface{}{record1, record2, record3}
   for _, record := range records {
       stream.Put(record)
   }
   ```

5. To ensure all records are sent and clean up resources, flush and stop streaming:

   ```go
   stream.FlushAndStopStreaming()
   ```

## Package Structure

The `inskinesis` package is organized as follows:

- `inskinesis` package: The main package containing the `StreamInterface`, `stream`, and related functionality for streaming records to Kinesis.
- `PartitionerFunction`: A customizable partitioning function for determining the partition key of records.
- `CreateBatches`: A utility function for creating batches of records.
- Various error handling and logging functionality.

## Usage

The package provides a simple interface for streaming records to a Kinesis stream. You can customize the configuration based on your needs, including the region, stream name, batch sizes, and partitioning function.

Here's an example of how to use the package:

```go
import "github.com/go-pkg/inskinesis"

config := inskinesis.Config{
    Region:                 "your-aws-region",
    StreamName:             "your-kinesis-stream-name",
    Partitioner:            nil, // Optionally provide a partitioner function
    MaxStreamBatchSize:     100, // Maximum size of each batch of records
    MaxStreamBatchByteSize: 1024 * 1024, // Maximum size in bytes for each batch
    MaxBatchSize:           500, // Maximum size of the log buffer
    MaxGroup:               10, // Maximum number of concurrent groups for sending records
}

stream, err := inskinesis.NewKinesis(config)
if err != nil {
    // Handle the error
}

// Send records to the stream
stream.Put(yourRecord)

// To ensure all records are sent and clean up resources, flush and stop streaming
stream.FlushAndStopStreaming()
```

## Error Handling

The `inskinesis` package provides error channels for receiving errors during streaming. You can use these channels to handle errors in your application gracefully. It's important to monitor the error channels to ensure the robustness of your data streaming process.

Here's an example of how to use the error channels:

```go
go func() {
   for {
      select {
      case err := <-stream.Error():
        sentry.Error(err)
      }
   }
}()

```

## Contributing

If you would like to contribute to the `inskinesis` package, please follow standard Go community guidelines for contributions. You can create issues, submit pull requests, and help improve the package for everyone.

## License

This package is available under the MIT License. You can find the full license details in the LICENSE file included in the package. Make sure to review and comply with the license when using the package in your project.
