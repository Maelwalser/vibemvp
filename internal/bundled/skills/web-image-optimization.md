# Web Image Optimization Skill Guide

## Next.js next/image

### Basic Usage

```tsx
import Image from 'next/image'

// Fixed dimensions
function Avatar({ src, name }: { src: string; name: string }) {
  return (
    <Image
      src={src}
      alt={name}
      width={64}
      height={64}
      className="rounded-full"
    />
  )
}
```

### Fill with Responsive Container

```tsx
// Parent must have position: relative and explicit dimensions
function HeroBanner() {
  return (
    <div className="relative w-full h-64 md:h-96">
      <Image
        src="/hero.jpg"
        alt="Hero banner"
        fill
        sizes="100vw"
        style={{ objectFit: 'cover' }}
        priority                      // LCP image — preloads eagerly
      />
    </div>
  )
}
```

### sizes Prop for Responsive Images

```tsx
function ProductCard({ image }: { image: string }) {
  return (
    <div className="relative aspect-square">
      <Image
        src={image}
        alt="Product"
        fill
        sizes="(max-width: 640px) 100vw, (max-width: 1024px) 50vw, 33vw"
        style={{ objectFit: 'contain' }}
      />
    </div>
  )
}
```

### Blur Placeholder

```tsx
import { getPlaiceholder } from 'plaiceholder'

// Server-side: generate blurDataURL from image
async function getImageProps(src: string) {
  const buffer = await fetch(src).then(r => r.arrayBuffer())
  const { base64 } = await getPlaiceholder(Buffer.from(buffer))
  return { blurDataURL: base64 }
}

// Component
<Image
  src="/photo.jpg"
  alt="Photo"
  width={800}
  height={600}
  placeholder="blur"
  blurDataURL="data:image/png;base64,..."   // tiny base64 preview
/>
```

### next.config.js — Remote Patterns

```js
// next.config.js
module.exports = {
  images: {
    remotePatterns: [
      { protocol: 'https', hostname: 'res.cloudinary.com', pathname: '/your-cloud/**' },
      { protocol: 'https', hostname: '**.amazonaws.com' },
    ],
    formats: ['image/avif', 'image/webp'],
    deviceSizes: [640, 750, 828, 1080, 1200, 1920],
    imageSizes: [16, 32, 48, 64, 96, 128, 256, 384],
  },
}
```

---

## Cloudinary

### URL Transformations

```ts
// Utility to build Cloudinary URLs
function cloudinaryUrl(
  publicId: string,
  transforms: Record<string, string | number> = {}
): string {
  const cloudName = process.env.NEXT_PUBLIC_CLOUDINARY_CLOUD_NAME
  const transformStr = Object.entries({
    f: 'auto',    // f_auto — best format (webp, avif, etc.)
    q: 'auto',    // q_auto — auto quality
    ...transforms,
  })
    .map(([k, v]) => `${k}_${v}`)
    .join(',')

  return `https://res.cloudinary.com/${cloudName}/image/upload/${transformStr}/${publicId}`
}

// Usage
const thumbnailUrl = cloudinaryUrl('products/chair', { w: 400, h: 400, c: 'fill' })
const heroUrl = cloudinaryUrl('banners/summer', { w: 1200, h: 400, c: 'fill', g: 'auto' })
// → https://res.cloudinary.com/mycloud/image/upload/f_auto,q_auto,w_400,h_400,c_fill/products/chair
```

### Upload with Preset

```ts
async function uploadToCloudinary(file: File): Promise<string> {
  const form = new FormData()
  form.append('file', file)
  form.append('upload_preset', 'ml_default')  // unsigned preset

  const res = await fetch(
    `https://api.cloudinary.com/v1_1/${process.env.NEXT_PUBLIC_CLOUDINARY_CLOUD_NAME}/image/upload`,
    { method: 'POST', body: form }
  )
  const data = await res.json()
  return data.public_id as string
}
```

### Signed URLs (Private Assets)

```ts
// Server-side only — never expose API secret to client
import { v2 as cloudinary } from 'cloudinary'

cloudinary.config({
  cloud_name: process.env.CLOUDINARY_CLOUD_NAME,
  api_key: process.env.CLOUDINARY_API_KEY,
  api_secret: process.env.CLOUDINARY_API_SECRET,
})

function getSignedUrl(publicId: string, expiresInSeconds = 3600): string {
  const timestamp = Math.round(Date.now() / 1000)
  const signature = cloudinary.utils.api_sign_request(
    { public_id: publicId, timestamp },
    process.env.CLOUDINARY_API_SECRET!
  )
  return `https://res.cloudinary.com/${process.env.CLOUDINARY_CLOUD_NAME}/image/upload` +
    `/s--${signature}--/f_auto,q_auto/${publicId}`
}
```

---

## Imgix

```ts
function imgixUrl(
  path: string,
  params: Record<string, string | number> = {}
): string {
  const base = `https://your-domain.imgix.net${path}`
  const query = new URLSearchParams({
    auto: 'format',   // serve webp/avif automatically
    fit: 'crop',
    ...Object.fromEntries(Object.entries(params).map(([k, v]) => [k, String(v)])),
  })
  return `${base}?${query}`
}

// Usage
const thumbUrl = imgixUrl('/products/chair.jpg', { w: 400, h: 400 })
const heroUrl  = imgixUrl('/banners/summer.jpg', { w: 1200, h: 400, crop: 'entropy' })
```

---

## Sharp (Self-Hosted Processing)

```ts
import sharp from 'sharp'
import path from 'path'
import fs from 'fs/promises'

interface ProcessImageOptions {
  width?: number
  height?: number
  quality?: number
  format?: 'webp' | 'jpeg' | 'avif' | 'png'
}

async function processImage(
  inputPath: string,
  outputPath: string,
  options: ProcessImageOptions = {}
): Promise<void> {
  const { width, height, quality = 80, format = 'webp' } = options

  let pipeline = sharp(inputPath)

  if (width || height) {
    pipeline = pipeline.resize(width, height, { fit: 'cover', position: 'attention' })
  }

  switch (format) {
    case 'webp':
      pipeline = pipeline.webp({ quality })
      break
    case 'avif':
      pipeline = pipeline.avif({ quality })
      break
    case 'jpeg':
      pipeline = pipeline.jpeg({ quality, mozjpeg: true })
      break
    case 'png':
      pipeline = pipeline.png({ compressionLevel: 8 })
      break
  }

  await pipeline.toFile(outputPath)
}

// Generate responsive image set
async function generateResponsiveSizes(inputPath: string, outputDir: string, name: string) {
  const sizes = [320, 640, 960, 1280, 1920]
  await Promise.all(
    sizes.map((w) =>
      processImage(inputPath, path.join(outputDir, `${name}-${w}.webp`), { width: w, format: 'webp' })
    )
  )
}

// In-memory transform (API route)
async function transformImageBuffer(buffer: Buffer): Promise<Buffer> {
  return sharp(buffer)
    .resize(800, 600, { fit: 'inside', withoutEnlargement: true })
    .webp({ quality: 82 })
    .toBuffer()
}
```

### Sharp in Next.js API Route

```ts
// pages/api/image-proxy.ts
import sharp from 'sharp'
import type { NextApiRequest, NextApiResponse } from 'next'

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  const { url } = req.query
  if (typeof url !== 'string') return res.status(400).end()

  const imageRes = await fetch(url)
  if (!imageRes.ok) return res.status(502).end()

  const buffer = Buffer.from(await imageRes.arrayBuffer())
  const processed = await sharp(buffer).webp({ quality: 80 }).toBuffer()

  res.setHeader('Content-Type', 'image/webp')
  res.setHeader('Cache-Control', 'public, max-age=31536000, immutable')
  res.send(processed)
}
```

---

## CDN Transform Patterns

```ts
// Bunny CDN / BunnyCDN
function bunnyCdnUrl(path: string, width: number, height?: number): string {
  return `https://your-zone.b-cdn.net${path}?width=${width}${height ? `&height=${height}` : ''}&quality=80`
}

// Supabase Storage Transform
function supabaseImageUrl(bucket: string, path: string, width: number): string {
  return `${process.env.NEXT_PUBLIC_SUPABASE_URL}/storage/v1/render/image/public/${bucket}/${path}?width=${width}&quality=80`
}

// Generic CDN query string pattern
function cdnTransform(baseUrl: string, params: { w?: number; h?: number; q?: number; f?: string }): string {
  const query = new URLSearchParams(
    Object.fromEntries(Object.entries(params).filter(([, v]) => v != null).map(([k, v]) => [k, String(v)]))
  )
  return `${baseUrl}?${query}`
}
```

---

## Key Rules

- Always set `priority` on the LCP (largest contentful paint) image — typically the hero or first viewport image.
- Always set `sizes` when using `fill` or variable-width images to prevent downloading oversized images.
- Use `f_auto` (Cloudinary) or `auto=format` (Imgix) to let the CDN serve the best format per browser.
- Never import Sharp in client-side code — it is Node.js only.
- Generate webp/avif alongside original formats for progressive enhancement.
- Set `Cache-Control: public, max-age=31536000, immutable` on transformed images with content-hashed URLs.
