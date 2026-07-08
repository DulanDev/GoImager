# GO IMAGER

Go-based microservice for resizing, converting, and optimizing images

## Features

- Resize images to specified dimensions
- Convert images between formats (JPEG, PNG, GIF)
- Optimize images for web use
- RESTful API for easy integration

## Quick Start

```sh
# Clone the repository
git clone https://github.com/yourusername/ImageProcessGo.git

# Navigate to the project directory
cd GoImager

# Install dependencies
go mod tidy

# Run the server
go run cmd/server/main.go
```

The server will start on `http://localhost:8080`.

## API Endpoints

- `POST /resize`: Resize an image
- `POST /convert`: Convert an image to a different format
- `POST /optimize`: Optimize an image for web use :: **TODO**

## Configuration

Environment variables:

- `PORT`: The port on which the server will run (default: 8080)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
