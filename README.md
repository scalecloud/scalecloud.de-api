# ScaleCloud.de API
**Backend for [www.scalecloud.de](https://www.scalecloud.de)**

The API powers ScaleCloud.de.

## Tech Stack

### Backend
- Go 1.27
- Gin
- Authentik (OpenID Connect)
- MongoDB Atlas
- Stripe
- SMTP
- Docker / Kubernetes

### Frontend
**[ScaleCloud.de](https://github.com/scalecloud/scalecloud.de)**

### Docker Image
[scalecloud.de-api](https://hub.docker.com/r/scalecloudde/scalecloud.de-api)

## Requirements

- Authentik
- MongoDB Atlas cluster and X.509 client certificate
- Stripe secret key and webhook signing secret
- SMTP server credentials
- `.env` configured from `.env.example`, including the frontend origin

## Run

Copy `.env.example` to `.env`, fill in the values, then run:

```sh
docker run -d --restart unless-stopped \
  -p 15000:15000 \
  --env-file=.env \
  --name scalecloud.de-api \
  scalecloudde/scalecloud.de-api:latest
```

## Health Checks

- `GET /health/live`: process status.
- `GET /health/ready`: dependency status; returns `503` if MongoDB, Stripe, or Authentik is unhealthy. SMTP is optional.

## SonarCloud Code Quality

<table>
  <thead>
    <tr>
      <th align="left">Category</th>
      <th align="center">Status</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>Quality</td>
      <td align="center">
        <img src="https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=alert_status&style=for-the-badge" alt="Quality Gate" />
        <img src="https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=bugs&style=flat-square" alt="Bugs" />
        <img src="https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=code_smells&style=flat-square" alt="Code Smells" />
      </td>
    </tr>
    <tr>
      <td>Security</td>
      <td align="center">
        <img src="https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=security_rating&style=flat-square" alt="Security Rating" />
        <img src="https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=vulnerabilities&style=flat-square" alt="Vulnerabilities" />
      </td>
    </tr>
    <tr>
      <td>Reliability</td>
      <td align="center">
        <a href="https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api">
          <img src="https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=reliability_rating&style=flat-square" alt="Reliability Rating" />
        </a>
      </td>
    </tr>
    <tr>
      <td>Maintainability</td>
      <td align="center">
        <img src="https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=sqale_rating&style=flat-square" alt="Maintainability" />
        <img src="https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=sqale_index&style=flat-square" alt="Technical Debt" />
      </td>
    </tr>
    <tr>
      <td>Coverage</td>
      <td align="center">
        <a href="https://sonarcloud.io/summary/new_code?id=scalecloud_scalecloud.de-api">
          <img src="https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=coverage&style=flat-square" alt="Coverage" />
        </a>
      </td>
    </tr>
    <tr>
      <td>Size</td>
      <td align="center">
        <img src="https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=duplicated_lines_density&style=flat-square" alt="Duplicated Lines" />
        <img src="https://sonarcloud.io/api/project_badges/measure?project=scalecloud_scalecloud.de-api&metric=ncloc&style=flat-square" alt="Lines of Code" />
      </td>
    </tr>
  </tbody>
</table>