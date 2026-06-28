## Notes

### Firebase iamPerms enum
`curl 'https://mobilesdk-pa.clients6.google.com/v1/projects/project-name-appspot-id:testIamPermissions?alt=json&key=KEY'` \
Get auth header from Firebase Console ? Anyways, there are better iam, this just stuck out because the request body had all the cool iam stuff

Post data: 
```json
{"permissions":["firebase.projects.own","firebase.projects.get","firebase.projects.update","firebase.projects.delete","resourcemanager.projects.update","resourcemanager.projects.delete","resourcemanager.projects.get","resourcemanager.projects.getIamPolicy","resourcemanager.projects.setIamPolicy","apikeys.keys.create","apikeys.keys.get","apikeys.keys.list","apikeys.keys.update","serviceusage.services.enable","serviceusage.services.get","firebase.clients.create","firebase.clients.delete","firebase.clients.get","firebase.clients.list","firebase.clients.undelete","firebase.clients.update","clientauthconfig.brands.create","clientauthconfig.brands.get","clientauthconfig.brands.update","clientauthconfig.clients.create","clientauthconfig.clients.delete","clientauthconfig.clients.get","clientauthconfig.clients.list","clientauthconfig.clients.update","oauthconfig.verification.get","firebase.billingPlans.get","firebase.billingPlans.update","resourcemanager.projects.createBillingAssignment","resourcemanager.projects.deleteBillingAssignment","firebaseapphosting.backends.create","firebaseapphosting.backends.delete","firebaseapphosting.backends.get","firebaseapphosting.backends.invoke","firebaseapphosting.backends.list","firebaseapphosting.backends.update","firebaseapphosting.builds.create","firebaseapphosting.builds.delete","firebaseapphosting.builds.get","firebaseapphosting.builds.list","firebaseapphosting.builds.update","firebaseapphosting.domains.create","firebaseapphosting.domains.delete","firebaseapphosting.domains.get","firebaseapphosting.domains.list","firebaseapphosting.domains.update","firebaseapphosting.locations.get","firebaseapphosting.locations.list","firebaseapphosting.operations.get","firebaseapphosting.operations.list","firebaseapphosting.operations.delete","firebaseapphosting.operations.cancel","firebaseapphosting.rollouts.create","firebaseapphosting.rollouts.delete","firebaseapphosting.rollouts.get","firebaseapphosting.rollouts.list","firebaseapphosting.rollouts.update","firebaseapphosting.traffic.get","firebaseapphosting.traffic.list","firebaseapphosting.traffic.update","firebaseauth.configs.create","firebaseauth.configs.get","firebaseauth.configs.getHashConfig","firebaseauth.configs.getSecret","firebaseauth.configs.update","firebaseauth.users.create","firebaseauth.users.get","firebaseauth.users.update","firebaseauth.users.delete","firebaseauth.users.sendEmail","firebasedatabase.instances.create","firebasedatabase.instances.get","firebasedatabase.instances.list","firebasedatabase.instances.update","firebasedatabase.instances.delete","datastore.databases.create","datastore.databases.delete","datastore.databases.getMetadata","datastore.databases.get","datastore.databases.list","datastore.databases.update","datastore.entities.get","datastore.entities.list","datastore.entities.create","datastore.entities.delete","datastore.entities.update","datastore.indexes.get","datastore.indexes.list","datastore.indexes.create","datastore.indexes.delete","datastore.indexes.update","datastore.backupSchedules.create","datastore.backupSchedules.delete","datastore.backupSchedules.list","datastore.backupSchedules.get","datastore.backupSchedules.update"]}
```


If identitytoolkit / identitytoolkit.googleapis.com is supported - https://www.googleapis.com/identitytoolkit/v3/relyingparty/getProjectConfig?key=AIzaKEY
```json
{
  "projectId": "101010101010101",
  "authorizedDomains": [
    "example-out-prod.firebaseapp.com",
    "example-out-prod.web.app",
    "exampleout.com",
    "pre-prod.exampleout.com",
    "output-example.redacted.com",
    "output-example-redacted.redacted.com",
    "mock.example.com",
    "localhost"
  ]
}
```

https://identitytoolkit.googleapis.com/v1/projects?key=
^ alias ? says it's legacy in the docs.

### Identitytoolkit Response Schema
https://identitytoolkit.googleapis.com/$discovery/rest
```json
 "IdentitytoolkitRelyingpartyGetProjectConfigResponse": {
  "id": "IdentitytoolkitRelyingpartyGetProjectConfigResponse",
  "description": "Response of getting the project configuration.",
  "type": "object",
  "properties": {
    "useEmailSending": {
      "description": "Whether to use email sending provided by Firebear.",
      "type": "boolean"
    },
    "changeEmailTemplate": {
      "$ref": "EmailTemplate",
      "description": "Change email template."
    },
    "projectId": {
      "type": "string",
      "description": "Project ID of the relying party."
    },
    "resetPasswordTemplate": {
      "$ref": "EmailTemplate",
      "description": "Reset password email template."
    },
    "authorizedDomains": {
      "type": "array",
      "description": "Authorized domains.",
      "items": {
        "type": "string"
      }
    },
    "legacyResetPasswordTemplate": {
      "$ref": "EmailTemplate",
      "description": "Legacy reset password email template."
    },
    "dynamicLinksDomain": {
      "type": "string"
    },
    "verifyEmailTemplate": {
      "$ref": "EmailTemplate",
      "description": "Verify email template."
    },
    "enableAnonymousUser": {
      "description": "Whether anonymous user is enabled.",
      "type": "boolean"
    },
    "apiKey": {
      "description": "Browser API key, needed when making http request to Apiary.",
      "type": "string"
    },
    "idpConfig": {
      "items": {
        "$ref": "IdpConfig"
      },
      "type": "array",
      "description": "OAuth2 provider configuration."
    },
    "allowPasswordUser": {
      "description": "Whether to allow password user sign in or sign up.",
      "type": "boolean"
    }
  }
}
```

### Uses
Could be used for extracting API key - project link regardless. If the key has access to the endpoint, we get the project id. If it doesn't... we get the project id anyways! I was looking for an endpoint that is always guaranteed to return the project id.

Other approaches rely on just error codes, could fail if the api key DOES have access to the method. Even Google Private APIs / -pa.googleapis would fail to give the project ID if the site you're testing is owned by google. Though... Most of those have referrer restrictions so it might still give it. Either way, in this case, both the error code and response include it, so it's perfect. No edge cases as far as I know ((?)).

### Responses & err codes
```json
{
  "error": {
    "code": 403,
    "message": "Requests to this API identitytoolkit method google.cloud.identitytoolkit.v1.ProjectConfigService.GetProjectConfig are blocked.",
    "errors": [
      {
        "message": "Requests to this API identitytoolkit method google.cloud.identitytoolkit.v1.ProjectConfigService.GetProjectConfig are blocked.",
        "domain": "global",
        "reason": "forbidden"
      }
    ],
    "status": "PERMISSION_DENIED",
    "details": [
      {
        "@type": "type.googleapis.com/google.rpc.ErrorInfo",
        "reason": "API_KEY_SERVICE_BLOCKED",
        "domain": "googleapis.com",
        "metadata": {
          "service": "identitytoolkit.googleapis.com",
          "methodName": "google.cloud.identitytoolkit.v1.ProjectConfigService.GetProjectConfig",
          "consumer": "projects/133333337",
          "apiName": "identitytoolkit"
        }
      },
      {
        "@type": "type.googleapis.com/google.rpc.LocalizedMessage",
        "locale": "en-US",
        "message": "Requests to this API identitytoolkit method google.cloud.identitytoolkit.v1.ProjectConfigService.GetProjectConfig are blocked."
      }
    ]
  }
}
```

## Thoughts...
This also means the key has  "https://www.googleapis.com/auth/cloud-platform" as a scope (most keys do though)

IAM Permission endpoint seems to be all over the place though. Look for RPC endpoint that can be standard.

On Google Cloud Storage, it's https://www.googleapis.com/storage/v1/b/bucket-name-here/iam/testPermissions?permissions=... but on ResourceManager it's on /v1/projects/PROJECT_ID_ALPHANUMERIC:testIamPermissions

https://gcp.permissions.cloud/ looks like a scraped IAM list. I dont think this is needed, as i have plenty of endpoints that list it. Might be more comprehensive, but this is pretty convenient for a starter.

Seems to be powered by https://github.com/iann0036/iam-dataset

```
map.json
A map of IAM permissions required for each method. [WORK IN PROGRESS]
```

I need a consistent IAM testperm endpoint, above all else. Seems like a good project though, might include a link to it in the README as a sorta "also check out...". Can't integrate it currently as it's not what I need. \
It seems to operate off the autogenerated Google Cloud docs, but those can be unreliable / outdated.

Service management, usagemetrics and discovery docs + proper firstparty, non-scraped iam sources will work better here. And will be more up to date.

GAPIs supports listing buckets by project https://storage.googleapis.com/storage/v1/b?project=901333333337&key=AIzaSy... \
^ can test if API key has access to list buckets. Huge if any API key gets this.

There is no IAM bleed between services. Attempt at accessing logging.buckets.get through GCS:
```json
{
  "error": {
    "code": 400,
    "message": "logging.buckets.get is not a valid Google Cloud Storage permission.",
    "errors": [
      {
        "message": "logging.buckets.get is not a valid Google Cloud Storage permission.",
        "domain": "global",
        "reason": "invalid"
      }
    ]
  }
}
```

Might be worthwile to look into all GCP services that support a project query param / numeric project ids. Could provide additional info IAM does not.

The Github thing includes a pretty flattened api listing with perms in methods_ext. Could use it to find the ?project params, but I could also do that through my own existing archive of Discovery documents. Most services are covered in mine, even internal ones, along with ~3 revisions per discovery doc.

IAM test perms seems to be a common RPC endpoint, available like so - google.storage.control.v2.StorageControl/TestIamPermissions. \
I used to have an endpoint that could call any RPC provided the auth was right. No idea where it is now.

The IAM RPC shows up in googleapis for servicemanagement

----

I'll be prioritizing alternative output formats once the major features are all implemented. That way I don't stall features while I make output formats for each one.

https://www.googleapis.com/storage/v1/b/cloud-samples-data/iam/testPermissions?permissions=storage.buckets.delete&permissions=storage.buckets.get&key=KEY

The cloud-samples-data bucket is widely used in Google documentation.\
 If one day that bucket retires and someone else claims it, they'd have bigger problems :P
https://www.googleapis.com/storage/v1/b/cloud-samples-data/iam/testPermissions?permissions=storage.buckets.delete&permissions=storage.buckets.get&key=KEY


The bucket name itself is not important. This API actually has a better use! It tells you if the API key is tied to an active billing account :)

```json
{
  "error": {
    "code": 403,
    "message": "The billing account for the owning project is disabled in state absent",
    "errors": [
      {
        "message": "The billing account for the owning project is disabled in state absent",
        "domain": "global",
        "reason": "accountDisabled",
        "locationType": "header",
        "location": "Authorization"
      }
    ]
  }
}```

Interestingly... It doesn't check for ACL for the specified bucket for the storage.buckets stuff, it checks the API key project instead. I'm not so sure for the non bucket* ones.

Some keys gave me:
```json
{
  "kind": "storage#testIamPermissionsResponse"
}
```

While some other keys gave me:
```json
{
  "kind": "storage#testIamPermissionsResponse",
  "permissions": [
    "storage.buckets.get"
  ]
}
```

Could be very useful. Will add with checks for all the diff permissions.

```json
  "TestIamPermissionsResponse": {
   "id": "TestIamPermissionsResponse",
   "type": "object",
   "description": "A storage.(buckets|objects|managedFolders).testIamPermissions response.",
   "properties": {
    "kind": {
     "type": "string",
     "description": "The kind of item this is.",
     "default": "storage#testIamPermissionsResponse"
    },
    "permissions": {
     "type": "array",
     "description": "The permissions held by the caller. Permissions are always of the format storage.resource.capability, where resource is one of buckets, objects, or managedFolders. The supported permissions are as follows:  \n- storage.buckets.delete - Delete bucket.  \n- storage.buckets.get - Read bucket metadata.  \n- storage.buckets.getIamPolicy - Read bucket IAM policy.  \n- storage.buckets.create - Create bucket.  \n- storage.buckets.list - List buckets.  \n- storage.buckets.setIamPolicy - Update bucket IAM policy.  \n- storage.buckets.update - Update bucket metadata.  \n- storage.objects.delete - Delete object.  \n- storage.objects.get - Read object data and metadata.  \n- storage.objects.getIamPolicy - Read object IAM policy.  \n- storage.objects.create - Create object.  \n- storage.objects.list - List objects.  \n- storage.objects.setIamPolicy - Update object IAM policy.  \n- storage.objects.update - Update object metadata. \n- storage.managedFolders.delete - Delete managed folder.  \n- storage.managedFolders.get - Read managed folder metadata.  \n- storage.managedFolders.getIamPolicy - Read managed folder IAM policy.  \n- storage.managedFolders.create - Create managed folder.  \n- storage.managedFolders.list - List managed folders.  \n- storage.managedFolders.setIamPolicy - Update managed folder IAM policy.",
     "items": {
      "type": "string"
     }
    }
   }
  }
```

### Perms list for storage googleapis
storage.buckets.delete
storage.buckets.get
storage.buckets.getIamPolicy
storage.buckets.create
storage.buckets.list
storage.buckets.setIamPolicy
storage.buckets.update
storage.objects.delete
storage.objects.get
storage.objects.getIamPolicy
storage.objects.create
storage.objects.list
storage.objects.setIamPolicy
storage.objects.update
storage.managedFolders.delete
storage.managedFolders.get
storage.managedFolders.getIamPolicy
storage.managedFolders.create
storage.managedFolders.list
storage.managedFolders.setIamPolicy

### Perm URL

> Note: Providing storage.buckets.list or storage.buckets.create returns an error, as these permissions apply to projects instead of buckets\
> *from https://docs.cloud.google.com/storage/docs/json_api/v1/buckets/testIamPermissions*

We'll test for .list and .create using the ?project= parameter & running their respective API stuff.

https://docs.cloud.google.com/storage/docs/json_api/v1/buckets/insert

https://www.googleapis.com/storage/v1/b/cloud-samples-data/iam/testPermissions?permissions=storage.buckets.delete&permissions=storage.buckets.get&permissions=storage.buckets.getIamPolicy&permissions=storage.buckets.create&permissions=storage.buckets.list&permissions=storage.buckets.setIamPolicy&permissions=storage.buckets.update&permissions=storage.objects.delete&permissions=storage.objects.get&permissions=storage.objects.getIamPolicy&permissions=storage.objects.create&permissions=storage.objects.list&permissions=storage.objects.setIamPolicy&permissions=storage.objects.update&permissions=storage.managedFolders.delete&permissions=storage.managedFolders.get&permissions=storage.managedFolders.getIamPolicy&permissions=storage.managedFolders.create&permissions=storage.managedFolders.list&permissions=storage.managedFolders.setIamPolicy&key=

Having any sort of perm could very well be indicative of GCS being accessible through it. The above link has all the perms, this one is restricted to the excluded one:
https://www.googleapis.com/storage/v1/b/cloud-samples-data/iam/testPermissions?permissions=storage.buckets.delete&permissions=storage.buckets.get&permissions=storage.buckets.getIamPolicy&permissions=storage.buckets.setIamPolicy&permissions=storage.buckets.update&permissions=storage.objects.delete&permissions=storage.objects.get&permissions=storage.objects.getIamPolicy&permissions=storage.objects.create&permissions=storage.objects.list&permissions=storage.objects.setIamPolicy&permissions=storage.objects.update&permissions=storage.managedFolders.delete&permissions=storage.managedFolders.get&permissions=storage.managedFolders.getIamPolicy&permissions=storage.managedFolders.create&permissions=storage.managedFolders.list&permissions=storage.managedFolders.setIamPolicy&key=

Interestingly, I get this error for the cloud-samples-data bucket
```json
{
  "error": {
    "code": 400,
    "message": "Cannot test storage.objects.getIamPolicy or storage.objects.setIamPolicy on buckets with uniform bucket-level access enabled",
    "errors": [
      {
        "message": "Cannot test storage.objects.getIamPolicy or storage.objects.setIamPolicy on buckets with uniform bucket-level access enabled",
        "domain": "global",
        "reason": "invalid"
      }
    ]
  }
}
```

Uniform bucket-level access disables ACLs (Access Control Lists) - "access to Cloud Storage resources then is granted exclusively through IAM"
Neat way to check for uniform bucket-level access on a specified bucket for post-scan scripts.

There's also another public bucket thatr DOESNT use uniform bucket level access. 
`https://www.googleapis.com/storage/v1/b/gcp-public-data-landsat/iam/testPermissions?permissions=storage.buckets.delete&permissions=storage.buckets.get&permissions=storage.buckets.getIamPolicy&permissions=storage.buckets.setIamPolicy&permissions=storage.buckets.update&permissions=storage.objects.delete&permissions=storage.objects.get&permissions=storage.objects.getIamPolicy&permissions=storage.objects.create&permissions=storage.objects.list&permissions=storage.objects.setIamPolicy&permissions=storage.objects.update&permissions=storage.managedFolders.delete&permissions=storage.managedFolders.get&permissions=storage.managedFolders.getIamPolicy&permissions=storage.managedFolders.create&permissions=storage.managedFolders.list&permissions=storage.managedFolders.setIamPolicy&key=`

I do think anything past buckets.* is pointless on a random bucket not belonging to the key. Still, the IAM testPermissions evidently can be used as a way to get some sort of additional data.

Referring back to this - https://storage.googleapis.com/storage/v1/b?project=PROJECT-ID-NUMBERS&key=AIza

When I checked it against my own project id and key, for a project that does not have GCS:
```json
{
  "error": {
    "code": 401,
    "message": "Requests to this API storage.googleapis.com method google.storage.buckets.list are blocked.",
    "errors": [
      {
        "message": "Requests to this API storage.googleapis.com method google.storage.buckets.list are blocked.",
        "domain": "global",
        "reason": "required",
        "locationType": "header",
        "location": "Authorization"
      }
    ]
  }
}
```

I don't have a billing account on GCP. If I did, my research would be greatly accelerated. Such is life. Maybe if my recent bugs pay out I might invest a good bit into my cybersec research. But if I get even more tea it would be just as good at helping me out in cybersec... too many options...

-----

## Enumerating .sandbox.google.com

Sandbox domains that exist report proper CSP headers.
If it doesn't exist, you don't get CSP headers. Can be used to validate the existence of .sandbox.google.com domains from source scans.

## Oauth

I still need a way to check the API key against an API. I'm also unaware of any GoogleAPIs / services that show `This API does not support API keys` for some endpoints while allowing it for others...

Actually, I've been looking for a commoon method for checking auth for so long, like a healthcheck, that I never checked the discovery doc. It seems the ones that require OAuth have a *top level* `auth` field. If I use the &fileds=auth I can get it to error out on those that don't have it, letting me keep using HEAD.

`https://bigquery.googleapis.com/$discovery/rest?key=AIzaSyD_CiIzuQlBIp_zXoQIecOI2s802lDkaGs&fields=auth`

### Top level auth

- admin.googleapis.com
- androidpublisher.googleapis.com
- androidenterprise.googleapis.com
- customsearch.googleapis.com
- calendar-json.googleapis.com
- bigquery.googleapis.com
- civicinfo.googleapis.com
- analytics.googleapis.com
- cloudkms.googleapis.com
- adsense.googleapis.com
- books.googleapis.com
- games.googleapis.com
- blogger.googleapis.com
- gmail.googleapis.com
- classroom.googleapis.com
- cloudresourcemanager.googleapis.com
- dataflow.googleapis.com
- discovery.googleapis.com
- doubleclicksearch.googleapis.com
- dfareporting.googleapis.com
- groupsmigration.googleapis.com
- people.googleapis.com
- pubsub.googleapis.com
- siteverification.googleapis.com
- shoppingcontent.googleapis.com
- tasks.googleapis.com
- tagmanager.googleapis.com
- ml.googleapis.com
- webfonts.googleapis.com
- reseller.googleapis.com
- youtube.googleapis.com
- sqladmin.googleapis.com
- toolresults.googleapis.com

There's no way that's everything? I know for a fact serviceusage exists in apis.go and also requires auth.
https://serviceusage.googleapis.com/$discovery/rest?fields=auth <- returns an error for some reason?? `"Error expanding 'fields' parameter. Cannot find matching fields for path 'auth'."`

I've just ran a normal scan with contents instead

APIs with top-level auth field. It shouldn't mean that it's not accessible at all. Lots of keys are granted these perms. I've called Servicemanagement from many API keys! This is just for prioritizing on quick runs.
These APIs explicitly define their authentication requirements in the discovery document
  - accessapproval.googleapis.com
  - accesscontextmanager.googleapis.com
  - addressvalidation.googleapis.com
  - adexchangebuyer.googleapis.com
  - admob.googleapis.com
  - adsdatahub.googleapis.com
  - adsense.googleapis.com
  - adsenseplatform.googleapis.com
  - advisorynotifications.googleapis.com
  - aiplatform.googleapis.com
  - airquality.googleapis.com
  - alertcenter.googleapis.com
  - alloydb.googleapis.com
  - analytics.googleapis.com
  - analyticsadmin.googleapis.com
  - analyticsdata.googleapis.com
  - analyticshub.googleapis.com
  - androidenterprise.googleapis.com
  - androidmanagement.googleapis.com
  - androidpublisher.googleapis.com
  - apigateway.googleapis.com
  - apigee.googleapis.com
  - apigeeregistry.googleapis.com
  - apihub.googleapis.com
  - apikeys.googleapis.com
  - apim.googleapis.com
  - appengine.googleapis.com
  - apphub.googleapis.com
  - areainsights.googleapis.com
  - artifactregistry.googleapis.com
  - assuredworkloads.googleapis.com
  - auditmanager.googleapis.com
  - authorizedbuyersmarketplace.googleapis.com
  - automl.googleapis.com
  - backupdr.googleapis.com
  - batch.googleapis.com
  - beyondcorp.googleapis.com
  - biglake.googleapis.com
  - bigquery.googleapis.com
  - bigqueryconnection.googleapis.com
  - bigquerydatapolicy.googleapis.com
  - bigquerydatatransfer.googleapis.com
  - bigquerymigration.googleapis.com
  - bigqueryreservation.googleapis.com
  - bigtable.googleapis.com
  - bigtableadmin.googleapis.com
  - billingbudgets.googleapis.com
  - binaryauthorization.googleapis.com
  - blockchainnodeengine.googleapis.com
  - blogger.googleapis.com
  - books.googleapis.com
  - calendar-json.googleapis.com
  - capacityplanner.googleapis.com
  - certificatemanager.googleapis.com
  - chat.googleapis.com
  - chromemanagement.googleapis.com
  - chromepolicy.googleapis.com
  - chromewebstore.googleapis.com
  - cloudasset.googleapis.com
  - cloudbilling.googleapis.com
  - cloudbuild.googleapis.com
  - cloudchannel.googleapis.com
  - cloudcommerceprocurement.googleapis.com
  - cloudcontrolspartner.googleapis.com
  - clouddeploy.googleapis.com
  - clouderrorreporting.googleapis.com
  - cloudfunctions.googleapis.com
  - cloudidentity.googleapis.com
  - cloudkms.googleapis.com
  - cloudlocationfinder.googleapis.com
  - cloudquotas.googleapis.com
  - cloudresourcemanager.googleapis.com
  - cloudscheduler.googleapis.com
  - cloudsearch.googleapis.com
  - cloudshell.googleapis.com
  - cloudsupport.googleapis.com
  - cloudtasks.googleapis.com
  - cloudtrace.googleapis.com
  - composer.googleapis.com
  - config.googleapis.com
  - connectors.googleapis.com
  - contactcenteraiplatform.googleapis.com
  - contactcenterinsights.googleapis.com
  - container.googleapis.com
  - containeranalysis.googleapis.com
  - contentwarehouse.googleapis.com
  - css.googleapis.com
  - datacatalog.googleapis.com
  - dataflow.googleapis.com
  - dataform.googleapis.com
  - datalabeling.googleapis.com
  - datalineage.googleapis.com
  - datamanager.googleapis.com
  - datamigration.googleapis.com
  - datapipelines.googleapis.com
  - dataplex.googleapis.com
  - dataportability.googleapis.com
  - dataproc.googleapis.com
  - datastore.googleapis.com
  - datastream.googleapis.com
  - deploymentmanager.googleapis.com
  - designcenter.googleapis.com
  - developerconnect.googleapis.com
  - dfareporting.googleapis.com
  - dialogflow.googleapis.com
  - discoveryengine.googleapis.com
  - displayvideo.googleapis.com
  - dlp.googleapis.com
  - dns.googleapis.com
  - docs.googleapis.com
  - documentai.googleapis.com
  - domains.googleapis.com
  - doubleclickbidmanager.googleapis.com
  - doubleclicksearch.googleapis.com
  - driveactivity.googleapis.com
  - drivelabels.googleapis.com
  - earthengine.googleapis.com
  - enterprisepurchasing.googleapis.com
  - essentialcontacts.googleapis.com
  - eventarc.googleapis.com
  - factchecktools.googleapis.com
  - fcm.googleapis.com
  - fcmdata.googleapis.com
  - file.googleapis.com
  - firebase.googleapis.com
  - firebaseappcheck.googleapis.com
  - firebaseappdistribution.googleapis.com
  - firebaseapphosting.googleapis.com
  - firebasedatabase.googleapis.com
  - firebasedataconnect.googleapis.com
  - firebasedynamiclinks.googleapis.com
  - firebasehosting.googleapis.com
  - firebaseml.googleapis.com
  - firebaseremoteconfig.googleapis.com
  - firebaserules.googleapis.com
  - firebasestorage.googleapis.com
  - firebasevertexai.googleapis.com
  - firestore.googleapis.com
  - forms.googleapis.com
  - games.googleapis.com
  - geminicloudassist.googleapis.com
  - gkebackup.googleapis.com
  - gkehub.googleapis.com
  - gkeonprem.googleapis.com
  - gmail.googleapis.com
  - gmailpostmastertools.googleapis.com
  - googleads.googleapis.com
  - groupsmigration.googleapis.com
  - groupssettings.googleapis.com
  - homegraph.googleapis.com
  - iam.googleapis.com
  - iamcredentials.googleapis.com
  - iap.googleapis.com
  - identitytoolkit.googleapis.com
  - ids.googleapis.com
  - indexing.googleapis.com
  - integrations.googleapis.com
  - jobs.googleapis.com
  - jules.googleapis.com
  - keep.googleapis.com
  - kmsinventory.googleapis.com
  - language.googleapis.com
  - licensing.googleapis.com
  - localservices.googleapis.com
  - logging.googleapis.com
  - looker.googleapis.com
  - lustre.googleapis.com
  - managedflink.googleapis.com
  - managedkafka.googleapis.com
  - manufacturers.googleapis.com
  - marketingplatformadmin.googleapis.com
  - meet.googleapis.com
  - memcache.googleapis.com
  - merchantapi.googleapis.com
  - metastore.googleapis.com
  - migrate.googleapis.com
  - migrationcenter.googleapis.com
  - modelarmor.googleapis.com
  - monitoring.googleapis.com
  - netapp.googleapis.com
  - networkconnectivity.googleapis.com
  - networkmanagement.googleapis.com
  - networksecurity.googleapis.com
  - notebooks.googleapis.com
  - observability.googleapis.com
  - ondemandscanning.googleapis.com
  - optimization.googleapis.com
  - oracledatabase.googleapis.com
  - orgpolicy.googleapis.com
  - oslogin.googleapis.com
  - parallelstore.googleapis.com
  - parametermanager.googleapis.com
  - paymentsresellersubscription.googleapis.com
  - people.googleapis.com
  - photoslibrary.googleapis.com
  - places.googleapis.com
  - playcustomapp.googleapis.com
  - playdeveloperreporting.googleapis.com
  - playintegrity.googleapis.com
  - policysimulator.googleapis.com
  - policytroubleshooter.googleapis.com
  - pollen.googleapis.com
  - privateca.googleapis.com
  - privilegedaccessmanager.googleapis.com
  - prod-tt-sasportal.googleapis.com
  - publicca.googleapis.com
  - pubsublite.googleapis.com
  - rapidmigrationassessment.googleapis.com
  - realtimebidding.googleapis.com
  - recaptchaenterprise.googleapis.com
  - recommendationengine.googleapis.com
  - recommender.googleapis.com
  - redis.googleapis.com
  - remotebuildexecution.googleapis.com
  - reseller.googleapis.com
  - retail.googleapis.com
  - run.googleapis.com
  - runtimeconfig.googleapis.com
  - saasservicemgmt.googleapis.com
  - sasportal.googleapis.com
  - script.googleapis.com
  - searchads360.googleapis.com
  - searchconsole.googleapis.com
  - secretmanager.googleapis.com
  - securesourcemanager.googleapis.com
  - securitycenter.googleapis.com
  - securityposture.googleapis.com
  - serviceconsumermanagement.googleapis.com
  - servicecontrol.googleapis.com
  - servicedirectory.googleapis.com
  - servicehealth.googleapis.com
  - servicemanagement.googleapis.com
  - servicenetworking.googleapis.com
  - serviceusage.googleapis.com
  - sheets.googleapis.com
  - siteverification.googleapis.com
  - slides.googleapis.com
  - smartdevicemanagement.googleapis.com
  - spanner.googleapis.com
  - speech.googleapis.com
  - sqladmin.googleapis.com
  - storage.googleapis.com
  - storagebatchoperations.googleapis.com
  - storageinsights.googleapis.com
  - storagetransfer.googleapis.com
  - streetviewpublish.googleapis.com
  - tagmanager.googleapis.com
  - tasks.googleapis.com
  - texttospeech.googleapis.com
  - tpu.googleapis.com
  - transcoder.googleapis.com
  - translate.googleapis.com
  - translationhub.googleapis.com
  - vault.googleapis.com
  - vectorsearch.googleapis.com
  - verifiedaccess.googleapis.com
  - videointelligence.googleapis.com
  - vision.googleapis.com
  - vmmigration.googleapis.com
  - vmwareengine.googleapis.com
  - vpcaccess.googleapis.com
  - walletobjects.googleapis.com
  - weather.googleapis.com
  - webrisk.googleapis.com
  - websecurityscanner.googleapis.com
  - workflowexecutions.googleapis.com
  - workflows.googleapis.com
  - workloadmanager.googleapis.com
  - workstations.googleapis.com
  - youtube.googleapis.com
  - youtubeanalytics.googleapis.com
  - youtubereporting.googleapis.com

No top level auth field
  - abusiveexperiencereport.googleapis.com
  - acceleratedmobilepageurl.googleapis.com
  - adexperiencereport.googleapis.com
  - admanager.googleapis.com
  - admin.googleapis.com
  - androiddeviceprovisioning.googleapis.com
  - androidovertheair.googleapis.com
  - businessprofileperformance.googleapis.com
  - checks.googleapis.com
  - civicinfo.googleapis.com
  - customsearch.googleapis.com
  - digitalassetlinks.googleapis.com
  - discovery.googleapis.com
  - firebaseinappmessaging.googleapis.com
  - generativelanguage.googleapis.com
  - kgsearch.googleapis.com
  - listallowedkids.googleapis.com
  - mapsbooking.googleapis.com
  - mybusinessaccountmanagement.googleapis.com
  - mybusinessbusinessinformation.googleapis.com
  - mybusinesslodging.googleapis.com
  - mybusinessnotifications.googleapis.com
  - mybusinessplaceactions.googleapis.com
  - mybusinessqanda.googleapis.com
  - mybusinessverifications.googleapis.com
  - partnerdev-mapsbooking.googleapis.com
  - photospicker.googleapis.com
  - playgrouping.googleapis.com
  - readerrevenuesubscriptionlinking.googleapis.com
  - regionlookup.googleapis.com
  - safebrowsing.googleapis.com
  - securetoken.googleapis.com
  - sts.googleapis.com
  - travelimpactmodel.googleapis.com
  - versionhistory.googleapis.com
  - webfonts.googleapis.com

I need to find a way to see which methods API keys are allowed to call without being bound to a service account. I know for a fact I can call servicemanagement with an API key, but not perform all actions.

Speaking of servicemanagement:

```json
"AuthenticationRule": {
  "id": "AuthenticationRule",
  "description": "Authentication rules for the service. By default, if a method has any authentication requirements, every request must include a valid credential matching one of the requirements. It's an error to include more than one kind of credential in a single request. If a method doesn't have any auth requirements, request credentials will be ignored.",
  "type": "object",
  "properties": {
    "selector": {
      "description": "Selects the methods to which this rule applies. Refer to selector for syntax details.",
      "type": "string"
    },
    "oauth": {
      "description": "The requirements for OAuth credentials.",
      "$ref": "OAuthRequirements"
    },
    "allowWithoutCredential": {
      "description": "If true, the service accepts API keys without any other credential. This flag only applies to HTTP and gRPC requests.",
      "type": "boolean"
    },
    "requirements": {
      "description": "Requirements for additional authentication providers.",
      "type": "array",
      "items": {
        "$ref": "AuthRequirement"
      }
    }
  }
}
```
Servicemanagement API can give all this.

```json
    "UsageRule": {
      "id": "UsageRule",
      "description": "Usage configuration rules for the service.",
      "type": "object",
      "properties": {
        "selector": {
          "description": "Selects the methods to which this rule applies. Use '*' to indicate all methods in all APIs. Refer to selector for syntax details.",
          "type": "string"
        },
        "allowUnregisteredCalls": {
          "description": "Use this rule to configure unregistered calls for the service. Unregistered calls are calls that do not contain consumer project identity. (Example: calls that do not contain an API key). WARNING: By default, API methods do not allow unregistered calls, and each method call must be identified by a consumer project identity.",
          "type": "boolean"
        },
        "skipServiceControl": {
          "description": "If true, the selected method should skip service control and the control plane features, such as quota and billing, will not be available. This flag is used by Google Cloud Endpoints to bypass checks for internal methods, such as service health check methods.",
          "type": "boolean"
        }
      }
    }
```

I'll do a deeper dive on it in a bit - https://servicemanagement.googleapis.com/$discovery/rest - I also have a feeling I can get a nice bug from servicemanagement.


curl 'https://developers.google.com/oauthplayground/getEndpoints' \
  -H 'Accept: application/json, text/javascript, */*; q=0.01' \
  -H 'Content-Type: application/json' \
  --data-raw '{"scopes":["https://www.googleapis.com/auth/cloud-platform","https://www.googleapis.com/auth/cloud-platform.read-only","https://www.googleapis.com/auth/service.management","https://www.googleapis.com/auth/servicecontrol"]}'

Intriguing

It's for oauth but it might help get an idea of what may be restricted, takes like 20 seconds for a server response tho.
The IAM API provides the roles provided in GCP and also the permissions per role. Might be useful to have a guess for the keys role.


