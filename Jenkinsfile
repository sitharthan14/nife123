pipeline {
    agent any
  stages {
    stage("Install Dependencies") {
      steps {
        echo env.BRANCH_NAME
        echo "------> Installing Dependencies <------"
        sh 'go mod tidy'
      }
    }
    stage("Build") {
      steps {
        echo "------> Building <------"
        sh 'pwd'
        sh 'go build'
      }
    }
    stage("Deploy") {
      steps {
          script {
              echo "------> Deploying <------"
              if(env.BRANCH_NAME=="develop")
              {
                  echo "------> Deploying to develop <------"
                  sh 'scp -o UserKnownHostsFile=/home/azureuser/known_hosts -i /home/azureuser/jenkins_key /var/lib/jenkins/workspace/Backend_Pipeline_develop/nife.io jenkins@34.93.16.182:/home/azureuser/nife.io/nife.io-new'
                  sh 'scp -o UserKnownHostsFile=/home/azureuser/known_hosts -i /home/azureuser/jenkins_key -r /var/lib/jenkins/workspace/Backend_Pipeline_develop/release/ jenkins@34.93.16.182:/home/azureuser/nife.io/'
                  sh 'cd /home/azureuser && pwd && ssh -o UserKnownHostsFile=/home/azureuser/known_hosts -i jenkins_key jenkins@34.93.16.182 \'bash -s\'<backend-deploy.sh'
//                   sh 'sudo ssh -i jenkins_key jenkins@34.93.16.182 \'bash -s\'<backend-deploy.sh'
              }
              if(env.BRANCH_NAME=="demo")
              {
                  echo "------> Deploying to demo <------"
                  sh 'scp -o UserKnownHostsFile=/home/azureuser/known_hosts -i /home/azureuser/jenkins_key /var/lib/jenkins/workspace/Backend_Pipeline_demo/nife.io jenkins@34.100.244.32:/home/azureuser/nife.io/nife.io-new'
                  sh 'scp -o UserKnownHostsFile=/home/azureuser/known_hosts -i /home/azureuser/jenkins_key -r /var/lib/jenkins/workspace/Backend_Pipeline_demo/release/ jenkins@34.100.244.32:/home/azureuser/nife.io/'
                  sh 'cd /home/azureuser && pwd && ssh -o UserKnownHostsFile=/home/azureuser/known_hosts -i jenkins_key jenkins@34.100.244.32 \'bash -s\'<backend-deploy.sh'
//                   sh 'sudo ssh -i jenkins_key jenkins@34.100.244.32 \'bash -s\'<backend-deploy.sh'
              }
              if(env.BRANCH_NAME=="prod")
              {
                  echo "------> Deploying to prod <------"
                  sh 'scp -o UserKnownHostsFile=/home/azureuser/known_hosts -i /home/azureuser/jenkins_key /var/lib/jenkins/workspace/Backend_Pipeline_prod/nife.io jenkins@34.93.142.76:/home/azureuser/nife.io/nife.io-new'
                  sh 'scp -o UserKnownHostsFile=/home/azureuser/known_hosts -i /home/azureuser/jenkins_key -r /var/lib/jenkins/workspace/Backend_Pipeline_prod/release/ jenkins@34.93.142.76:/home/azureuser/nife.io/'
                  sh 'cd /home/azureuser && pwd && ssh -o UserKnownHostsFile=/home/azureuser/known_hosts -i jenkins_key jenkins@34.93.142.76 \'bash -s\'<backend-deploy.sh'
//                   sh 'sudo ssh -i jenkins_key jenkins@34.93.142.76 \'bash -s\'<backend-deploy.sh'
              }
          }
      }
    }
  }
}
