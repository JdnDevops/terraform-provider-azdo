# JDN AZDO Provider (Terraform Plugin Framework)

_This template repository is built on the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework). The template repository built on the [Terraform Plugin SDK](https://github.com/hashicorp/terraform-plugin-sdk) can be found at [terraform-provider-scaffolding](https://github.com/hashicorp/terraform-provider-scaffolding). See [Which SDK Should I Use?](https://developer.hashicorp.com/terraform/plugin/framework-benefits) in the Terraform documentation for additional information._

This is a custom provider, using the Azure Devops SDK for GO, to get the identities from Azure Devops.
Currently the Azure Devops provider does not include the needed attributes in their implemation of de Identity datasource.
For our case we also needed the subject descriptor.

This provider is made for Azure Devops 2022. We wanted to apply git permission and branch policies. But these required either an Identity descriptor or Identity subject descriptor. The Azure Devops provider uses the Graph api the get groups, but this is not available on Azure Devops 2022. So we create our own provider to get the Groups with the needed attributes.