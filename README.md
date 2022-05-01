# API for Scalecloud.de

Hello I'm the backend for www.scalecloud.de.

Frontent: https://github.com/scalecloud/scalecloud.de  
Auth: Firebase  
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
    --mount type=bind,source="<keys-dir>",destination=/app/keys \
    --log-driver local --log-opt max-size=100m --log-opt max-file=2 \
    --name scalecloud.de-api scalecloudde/scalecloud.de-api:latest
```

### Folder `<keys-dir>`

The following files are required in the `<keys-dir>` folder:  
Firebase: `firebase-serviceAccountKey.json`  
Mongodb-Atlas: `mongodb-atlas.pem`  
Stripe: `stripe-secret-key.json`  

## SonarCloud.io

[![Bugs](https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=bugs)](https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api)
[![Code Smells](https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=code_smells)](https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=coverage)](https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api)
[![Duplicated Lines (%)](https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=duplicated_lines_density)](https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api)
[![Lines of Code](https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=ncloc)](https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api)
[![Technical Debt](https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=sqale_index)](https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=vulnerabilities)](https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api)
