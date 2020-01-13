# pfxsigner
pfxsigner is a utility (CLI) and an HTTP server for digitally signing PDFs with signatures loaded from PFX (PKCS #12) certificates. It can load multiple named certificates from PFX files and sign PDFs with them.

## Configuration
The signature properties are recorded in a JSON file. See `props.json.sample`.

Multiple PFX files can be loaded by specifying the `-pfx certname|/cert/path.pfx|certpassword` param multiple times.

## CLI
The CLI mode supports multi-threaded bulk-signing of PDFs.

```shell
# Pipe list of documents to convert to stdin. Each line should be in the format src-doc.pdf|signed-doc.pdf
# eg:
# mycert|a.pdf|a-signed.pdf
# mycert|b.pdf|b-signed.pdf
echo "in.pdf|out.pdf" | ./pfxsigner -pfx "mycert|/path/cert.pfx|certpass" -props-file "props.json.sample" cli -workers 4
```

## Server
In the server mode, pfxsigner exposes an HTTP API to which a PDF file and signature properties (optional) can be posted to received a signed PDF.

```shell
# Start the server
./pfxsigner -pfx "mycert|/path/cert.pfx|certpass" -props-file "props.json" server
```

```shell
# Sign a pdf

REQ=$(cat props.json.sample)
curl -F "props=$REQ" -F 'file=@./test.pdf' -o './test-signed.pdf' localhost:8000/document
```

## Docker

You can use the [official]() Docker image to run `pfxsigner`.

**NOTE**: You'll need to mount `cert.pfx` and `props.json` from a directory available on host machine to a directory inside container. You can do that by passing `-v </path/on/host>:</path/on/container>` while launching the container.

```shell
# For example `./data` contains `cert.pfx` and `props.json`.
export PFX_PASSWORD=mysecurepass
docker run -it -p 8000:8000 -v "$PWD"/data:/data kailashnadh/pfxsigner:latest -pfx-file /data/cert.pfx  -pfx-password $PFX_PASSWORD -props-file /data/props.json server
```

### API
The API endpoint is `:8000/document`. It accepts a POST request (multipart/form-data) with the following fields.

| Field   |                                                               |
|---------|---------------------------------------------------------------|
| `props` | Signature properties as a JSON string (see props.json.sample). If not set, the default properties loaded during runtime are used |
| `file`  | The PDF file to sign                                          |

## License

pfxsigner is licensed under the AGPL v3 license.
