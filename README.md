# AKS Workload Identity ADAL Bridge

This application is designed to bridge the gap between traditional MSI authentication and workload identity as used in Azure Kubernetes Service.

If you have a service or tool that only supports ADAL-based MSI authentication (and not OIDC), but you use Workload Identity and don't want to provision static credentials to work around this incompatibility, this tool may be for you.

It is designed to be run in a sidecar container for your app running in AKS, and mimics the [Instance Metadata Service](https://learn.microsoft.com/en-us/azure/virtual-machines/instance-metadata-service) (IMDS) to support MSI authentication.

## Prerequisites

- Workload Identity enabled in your AKS cluster along with the OIDC issuer
- A pod with the `azure.workload.identity/use: "true"` label

- The ability to override the default `http://169.254.169.254` endpoint (i.e. IMDS) for your application/tool/SDK

## Usage

To run this tool as a sidecar container, add the following to your pod spec:

```yaml
initContainers:
  - name: workload-identity-adal-bridge
    image: "manicminer/workload-identity-adal-bridge:0.4.1"
    restartPolicy: Always
    env:
      - name: LOG_LEVEL
        value: "info"
```

The bridge tool can then be used in one of two ways:

### Override the instance metadata host/URL

Typically by exporting a special environment variable (which unfortunately differs across languages and SDKs), you override the default `http://169.254.169.254` endpoint (i.e. IMDS) for your application, so that it queries this tool instead.

Known environment variables include:

- `IMDS_ENDPOINT`
- `MSI_ENDPOINT`

The service will use the value of the `AZURE_CLIENT_ID` environment variable, as set by Workload Identity, but you can override this by instructing your application to include the `clientId` parameter when querying the emulated IMDS API.

### Use the bundled client to output an access token

If you are unable to redirect your application to query the API provided by this tool, you can use the included client to query it yourself and output an access token for your application to use. This method is more involved as you will need to consider token expiration and renewal, and it may be difficult to inject the token into your application.

However, a notable example of where this could be the easier option is the [`bcp` utility](https://learn.microsoft.com/en-us/sql/tools/bcp-utility) included in MSSQL Tools.

To use this method, you may wish to include the `workload-identity-adal-bridge` binary in your main container image, for example:

```dockerfile
ADD --chmod=755 https://github.com/manicminer/workload-identity-adal-bridge/releases/download/v0.4.1/workload-identity-adal-bridge-linux-amd64 /usr/local/bin/workload-identity-adal-bridge
```

Then in your entrypoint script, you might invoke the client to acquire an access token like this:

```bash
token="$(workload-identity-adal-bridge client token --resource "https://database.windows.net")"
if [ $? != 0 ]; then
  echo "Failed to acquire access token" >&2
  exit 1
fi
```

Note that the `resource` must be specified, [see docs for more information](https://learn.microsoft.com/en-us/entra/identity/managed-identities-azure-resources/how-to-use-vm-token#get-a-token-using-http). You can get a list of common resources by looking at [metadata20220901.json](https://github.com/manicminer/workload-identity-adal-bridge/blob/main/metadata/metadata20220901.json) or reading through your SDK of choice.

How you deliver the access token to your application depends entirely on your application, but here's an example for [MSSQL `bcp`](https://learn.microsoft.com/en-us/sql/tools/bcp-utility):

```bash
echo -n "${token}" | iconv -f ascii -t UTF-16LE >/tmp/bcp-token
/opt/mssql-tools/bin/bcp "table_name" in "data.tab" -S "${DB_HOST}" -d "${DB_NAME}" -G -P /tmp/bcp-token -F1 -f "data.fmt" -b 100
```
