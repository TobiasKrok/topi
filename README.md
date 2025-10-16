# Topi - toby's pipelines

This is a learning project for me to learn Golang and Kubernetes operators. It is supposed to be something like a GitHub Actions/Azure Pipelines project.
Some Claude has been used for suggestions on architecture and documentation

## Description

Topi is an Azure Pipelines-like CI/CD sysSem consisting of three components:

- **Engine**: HTTP server that listens to Git webhooks
- **Scheduler**: Kubernetes operator that manages build jobs
- **Builder**: Application that builds projects based on workflows
