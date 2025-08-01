# Foundry API Scripts

This directory contains scripts for managing the Foundry API infrastructure.

## init.sh

The `init.sh` script is responsible for generating and uploading authentication credentials to AWS Secrets Manager for the Foundry API and Operator services.

### What it does

1. **Generates certificates**: Uses Earthly to generate public and private key pairs for API authentication
2. **Uploads certificates to AWS Secrets Manager**: Stores the certificates in the secret `FOUNDRY_API_CERTS_SECRET` with the following structure:
   ```json
   {
     "public.pem": "-----BEGIN PUBLIC KEY-----...",
     "private.pem": "-----BEGIN PRIVATE KEY-----..."
   }
   ```
3. **Generates operator token**: Uses Earthly to generate a JWT token for the Foundry Operator
4. **Uploads operator token to AWS Secrets Manager**: Stores the token in the secret `FOUNDRY_OPERATOR_TOKEN_SECRET` with the following structure:
   ```json
   {
     "token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
   }
   ```

### Prerequisites

- AWS CLI configured with appropriate permissions
- Earthly installed and configured
- `jq` command-line tool installed
- AWS region set via `AWS_REGION` environment variable (defaults to `eu-central-1`)

### Environment Variables

The script requires the following environment variables to be set:

- `FOUNDRY_API_CERTS_SECRET` - AWS Secrets Manager path for API certificates
- `FOUNDRY_OPERATOR_TOKEN_SECRET` - AWS Secrets Manager path for operator JWT token

You can set these variables in a `.env` file in the same directory as the script. Copy `env.example` to `.env` and update the values:

```bash
cp env.example .env
# Edit .env with your actual secret paths
```

### Usage

```bash
./init.sh
```

### Security

- The script automatically cleans up local certificate and token files after upload
- Sensitive files are removed from the local filesystem using a trap that runs on script exit
- All credentials are stored securely in AWS Secrets Manager

### AWS Secrets Created/Updated

The script creates or updates secrets at the paths specified in your environment variables:

- `$FOUNDRY_API_CERTS_SECRET` - Contains API authentication certificates
- `$FOUNDRY_OPERATOR_TOKEN_SECRET` - Contains operator JWT token