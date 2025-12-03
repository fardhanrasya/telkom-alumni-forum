upload file image:
You can customize your upload by passing additional parameters in the options object. This allows you to assign metadata, organize assets, reuse filenames, request moderation, and more. For more details, see Customizing uploads.

Example: Upload with tags, metadata, moderation, and analysis

This example sets:

Tags: summer, new-arrival

Contextual metadata:

Set the department as apparel
Set the photographer as Jane Doe
Structured metadata:

Set the field with external ID sku-id as SKU12345678
Set the field with external ID product-id as PROD-9081-WHT
Additional options:

use_filename: true – use original file name as the base for public ID
unique_filename: true – append a random suffix to avoid overwriting
moderation: "webpurify" – automatically flag assets for moderation
quality_analysis: true – request Cloudinary’s AI quality scoring

    resp, err := cld.Upload.Upload(ctx, "/home/my_image.jpg", uploader.UploadParams{
        UseFilename:     api.Bool(true),
        UniqueFilename:  api.Bool(true),
        Moderation:      "webpurify",
        Tags:            []string{"summer", "new-arrival"},
        Context: map[string]string{
            "department":  "apparel",
            "photographer": "Jane Doe",
        },
        Metadata: map[string]string{
            "sku-id":     "SKU12345678",
            "product-id": "PROD-9081-WHT",
        },
        QualityAnalysis: api.Bool(true),
    })

    if err != nil {
        log.Fatalf("Upload failed: %v", err)
    }

    log.Printf("Upload successful: %s", resp.SecureURL)

By default, uploading is performed synchronously. Once finished, the uploaded image or video is immediately available for transformation and delivery. An upload call returns a struct with content similar to the following: 
&{
  AssetID:aac8fd89108ff834dd27f979fd9ce77e 
  PublicID:hl22acprlomnycgiudor 
  Version:1591095352 
  VersionID:909700634231dbaaf8b06d7a5940299e
  Signature:86922996d63e596464ea3d7a5e86e8de8123f23f 
  Width:1200 
  Height:1200 
  Format:jpg 
  ResourceType:image 
  CreatedAt:2020-06-02 10:55:52 +0000 UTC 
  Tags:[] 
  Pages:0 
  Bytes:460268 
  Type:upload 
  Etag:2c7e88604ba3f340a0c5bc8cd418b4d9 
  Placeholder:false 
  URL:  http://res.cloudinary.com/demo/image/upload/v1591095352/hl22acprlomnycgiudor.jpg 
  SecureURL:  https://res.cloudinary.com/demo/image/upload/v1591095352/hl22acprlomnycgiudor.jpg
  AssetFolder: ,
  DisplayName:do8wnccnlzrfvwv1mqkq 
  Overwritten:true 
  OriginalFilename:my_image 
  ApiKey:614335564976464
  Error:{Message:}
}

json:
{
    "asset_id": "86ca8ba13b17e21d23534b7e842b8847",
    "public_id": "do8wnccnlzrfvwv1mqkq",
    "version": 1719309138,
    "version_id": "1a2b0a8ef0bf8e9f20a922f38704eda6",
    "signature": "afb6a3374ba12e4e0307e23e625d939b242ddb5c",
    "width": 1920,
    "height": 1281,
    "format": "jpg",
    "resource_type": "image",
    "created_at": "2024-06-25T09:52:18Z",
    "tags": [],
    "bytes": 310479,
    "type": "upload",
    "etag": "a8f8236455d352b8cee6aba0e3fbc87e",
    "placeholder": false,
    "url": "http://res.cloudinary.com/cld-docs/image/upload/v1719309138/do8wnccnlzrfvwv1mqkq.jpg",
    "secure_url": "https://res.cloudinary.com/cld-docs/image/upload/v1719309138/do8wnccnlzrfvwv1mqkq.jpg",
    "asset_folder": "",
    "display_name": "do8wnccnlzrfvwv1mqkq",
    "original_filename": "f5lq8lfq8pfj0xmd9dak",
    "api_key": "614335564976464"
}
The response includes HTTP and HTTPS URLs for accessing the uploaded media asset as well as additional information regarding the uploaded asset: The public ID, resource type, width and height, file format, file size in bytes, a signature for verifying the response and more.

