# pfxsigner
pfxsigner is a utility (CLI) and an HTTP server for digitally signing PDFs with signatures loaded from PFX (PKCS #12) certificates.

## Configuration
The signature properties are recorded in a JSON file. See `props.json.sample`.

## CLI
The CLI mode supports multi-threaded bulk-signing of PDFs.

```shell
# Pipe list of documents to convert to stdin. Each line should be in the format src-doc.pdf|signed-doc.pdf
# eg:
# a.pdf|a-signed.pdf
# b.pdf|b-signed.pdf
echo "in.pdf|out.pdf" | ./pfxsigner -pfx-file cert.pfx -props-file "props.json.sample" cli -workers 4
```

## Server
In the server mode, pfxsigner exposes an HTTP API to which a PDF file and signature properties (optional) can be posted to received a signed PDF.

```shell
# Start the server
./pfxsigner -pfx-file cert.pfx -props-file "props.json" server
```

```shell
# Sign a pdf

REQ=$(cat props.json.sample)
curl -F "props=$REQ" -F 'file=@./test.pdf' -o './test-signed.pdf' localhost:8000/document
```

### API
The API endpoint is `:8000/document`. It accepts a POST request (multipart/form-data) with the following fields.

| Field   |                                                               |
|---------|---------------------------------------------------------------|
| `props` | Signature properties as a JSON string (see props.json.sample). If not set, the default properties loaded during runtime are used |
| `file`  | The PDF file to sign                                          |

## License

pfxsigner is licensed under the AGPL v3 license.
