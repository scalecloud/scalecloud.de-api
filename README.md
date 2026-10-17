# API for Scalecloud.de

Hello I'm the backend for www.scalecloud.de.

Frontent: https://github.com/scalecloud/scalecloud.de  
Auth: Authentik (OpenID Connect)
NoSQL: MongoDB-Atlas  
Payment: Stripe  

## Running the API in Docker

```
docker pull scalecloudde/scalecloud.de-api:latest
```
<https://hub.docker.com/r/scalecloudde/scalecloud.de-api>

```
docker run -d --restart unless-stopped \
    -p 15000:15000 \
  -e OIDC_ISSUER="https://auth.example.com/application/o/<provider-slug>/" \
  -e OIDC_CLIENT_ID="<provider-client-id>" \
    --mount type=bind,source="<keys-dir>",destination=/app/keys \
    --log-driver local --log-opt max-size=100m --log-opt max-file=2 \
    --name scalecloud.de-api scalecloudde/scalecloud.de-api:latest
```

### Folder `<keys-dir>`

The following files are required in the `<keys-dir>` folder:

- Firebase Admin (used by the current seat-invitation flow): `firebase-serviceAccountKey.json`
- MongoDB-Atlas X.509 Certificate: `mongodb-atlas.pem`
- MongoDB-Atlas connection string: `mongodb-atlas-connection-string.txt`
- Stripe: `stripe-secret-key.txt`
- SMTP: `smtp-credentials.json`

Set `OIDC_ISSUER` to the exact Authentik provider issuer URL (including the provider slug and trailing slash), and `OIDC_CLIENT_ID` to that provider's client ID. The API discovers the provider configuration at startup and verifies access-token signatures and claims using its JWKS; it does not call Authentik for every request. The token's `sub` claim is used as the user UID, so keep the Authentik subject stable and migrate existing Firebase UID records before changing the subject format. Request the `email` scope and include an email claim in the access token for API operations that create Stripe customers.

Seat invitations still use Firebase Admin and store Firebase UIDs. They will not automatically match Authentik `sub` values; migrate that invitation/account-linking flow before relying on invited seats with Authentik accounts.

#### SMTP Credentials File (`smtp-credentials.json`)

The `smtp-credentials.json` file should contain the following structure:

Example:

```json
{
  "host": "smtp.example.com",
  "port": 587,
  "username": "user@example.com",
  "password": "yourpassword",
  "from": "noreply@example.com"
}
```

## SonarCloud.io

[![Bugs](https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=bugs)](https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api)
[![Code Smells](https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=code_smells)](https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api)
[![Duplicated Lines (%)](https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=duplicated_lines_density)](https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api)
[![Lines of Code](https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=ncloc)](https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api)
[![Technical Debt](https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=sqale_index)](https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=vulnerabilities)](https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api)