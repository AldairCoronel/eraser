# Eraser: Cleaning up Images from Kubernetes Nodes

Eraser helps Kubernetes admins remove a list of non-running images from all Kubernetes nodes in a cluster.

## Design 
* [Design Documentation](https://docs.google.com/document/d/1Rz1bkZKZSLVMjC_w8WLASPDUjfU80tjV-XWUXZ8vq3I/edit?usp=sharing) 

## Getting started

Create an [ImageList](config/samples/eraser_v1alpha1_imagelist.yaml) and specify the images you would like to remove manually.

* `kubectl apply -f config/samples/eraser_v1alpha1_imagelist.yaml --namespace=eraser-system`

This should have triggered an [ImageJob](api/v1alpha1/imagejob_types.go) that will deploy [eraser](pkg/eraser/eraser.go) pods on every node to perform the removal given the list of images. 

To view the result of the removal:
* describe ImageList CR and look at status field:
    * `kubectl describe ImageList -n eraser-system imagelist_sample`

To view the result of the ImageJob eraser pods:
* find name of ImageJob: 
    * `kubectl get ImageJob -n eraser-system`
* describe ImageJob CR and look at status field:
    * `kubectl describe ImageJob -n eraser-system [name of ImageJob]`

## Developer Setup

Developing this project requires access to a Kubernetes cluster and Go version 1.16 or later.

## Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.opensource.microsoft.com.

When you submit a pull request, a CLA bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., status check, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

## Trademarks

This project may contain trademarks or logos for projects, products, or services. Authorized use of Microsoft 
trademarks or logos is subject to and must follow 
[Microsoft's Trademark & Brand Guidelines](https://www.microsoft.com/en-us/legal/intellectualproperty/trademarks/usage/general).
Use of Microsoft trademarks or logos in modified versions of this project must not cause confusion or imply Microsoft sponsorship.
Any use of third-party trademarks or logos are subject to those third-party's policies.
