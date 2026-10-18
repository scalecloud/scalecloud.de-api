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
  -e OIDC_ISSUER="https://auth.scalecloud.de/application/o/<provider-slug>/" \
  -e OIDC_CLIENT_ID="<provider-client-id>" \
  -e AUTHENTIK_API_URL="https://auth.scalecloud.de/api/v3/" \
  -e AUTHENTIK_API_TOKEN="<server-side-api-token>" \
  -e INVITATION_CLAIM_URL="https://www.scalecloud.de/claim-invite" \
    --mount type=bind,source="<keys-dir>",destination=/app/keys \
    --log-driver local --log-opt max-size=100m --log-opt max-file=2 \
    --name scalecloud.de-api scalecloudde/scalecloud.de-api:latest
```

### Folder `<keys-dir>`

The following files are required in the `<keys-dir>` folder:

- MongoDB-Atlas X.509 Certificate: `mongodb-atlas.pem`
- MongoDB-Atlas connection string: `mongodb-atlas-connection-string.txt`
- Stripe: `stripe-secret-key.txt`
- SMTP: `smtp-credentials.json`

Set `OIDC_ISSUER` to the exact Authentik provider issuer URL (including the provider slug and trailing slash), and `OIDC_CLIENT_ID` to that provider's client ID. The API discovers the provider configuration at startup and verifies access-token signatures and claims using its JWKS; it does not call Authentik for every request. The token's `sub` claim is used as the user UID. Request the `email` scope and include `email` and `email_verified` claims in the access token.

`AUTHENTIK_API_URL` is the Authentik API v3 base URL. Create a server-side API token with permission to view users, create and view groups, and add/remove group members; never expose it to Angular. `INVITATION_CLAIM_URL` is the public Angular route that accepts invite tokens. The API stores pending invitations in MongoDB, expires them after seven days, and creates one Authentik group named `scalecloud-subscription-<stripe-subscription-id>` for each subscription as its first seat is provisioned. Active seat membership is synchronized to that group and removed when a seat or subscription is deleted.

The Angular invite page calls `POST /invitation/preview` with the token to show safe invitation details. After Authentik login, it calls `POST /invitation/claim` with the same token and the normal Bearer access token. The API requires a verified email matching the invitation before it activates the MongoDB seat and adds the Authentik user to the subscription group.

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