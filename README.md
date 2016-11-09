HTTPSink
======================

Capture all requests that come into the server and be able to get those requests later for examination. Useful for faking API endpoints.

Using configuration or environment variables, you can set the foreign API endpoint in your code to use "localhost:8888" (or any port). Now, in tests, you can use httpsink to stand up an HTTP sink that will either just swallow up your requests or you can choose for it to yield a desired response.

## Features

- Configurable port and interface.
- Inspect the requests that come into the sink via the sink `GET /request/[index]` endpoint where `[index]` is the one-indexed request to come into the sink. You can also get the last request by calling `GET /requests/latest`.
- Set a capacity for the total number of requests that the sink will allow before it rejects them.
- Clear stored requests via `DELETE /requests`
- Run multiple sinks at the same time.

## Usage

See [example.go](example.go).
