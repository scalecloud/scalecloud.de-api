def lastStage = ''
node() {
  properties([disableConcurrentBuilds()])
  try {
    currentBuild.result = "SUCCESS"

    stage('Clean Workspace') {
     cleanWs()
    }
    stage('Checkout') {
      lastStage = env.STAGE_NAME
      checkout scm
      echo "Current build result: ${currentBuild.result}"
    }
    stage('Build and Test') {
        lastStage = env.STAGE_NAME
        sh "docker build --tag scalecloudde/scalecloud.de-api:beta ."
    }
    stage('Push Docker Image') {
        lastStage = env.STAGE_NAME
        sh 'docker push scalecloudde/scalecloud.de-api:beta'
    }
  }
  catch (err) {
    echo "Caught errors! ${err}"
    echo "Setting build result to FAILURE"
    currentBuild.result = "FAILURE"

    mail bcc: '', body: "<br>Project: ${env.JOB_NAME} <br>Build Number: ${env.BUILD_NUMBER} <br> URL de build: ${env.BUILD_URL}", cc: '', charset: 'UTF-8', from: '', mimeType: 'text/html', replyTo: '', subject: "ERROR CI: Project name -> ${env.JOB_NAME}", to: "jenkins@scalecloud.de";
       
    throw err
  }
  finally {
    stage('Clean Workspace') {
      cleanWs()
    }
    stage('Docker remove not needed images') {
        sh 'docker rmi golang:1.19'
        sh 'docker rmi gcr.io/distroless/base-debian11:latest'
    }
  }
}