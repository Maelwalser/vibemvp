# Web Accessibility & SEO Skill Guide

## Accessibility (A11y)

### Color Contrast

```ts
// WCAG 2.1 requirements
// AA: 4.5:1 for normal text, 3:1 for large text (18pt / 14pt bold)
// AAA: 7:1 for normal text, 4.5:1 for large text

// Check contrast with postcss-plugin-autoprefix or via Tailwind config
// tailwind.config.js — enforce via ESLint plugin
// "eslint-plugin-jsx-a11y" checks many a11y rules automatically
```

```css
/* Good contrast examples */
.text-primary { color: #1f2937; background: #ffffff; }   /* 16:1 */
.btn-primary  { color: #ffffff; background: #1d4ed8; }   /* 5.9:1 */
.text-muted   { color: #6b7280; background: #ffffff; }   /* 4.6:1 — barely AA */

/* Bad: fails AA */
/* .text-light { color: #9ca3af; background: #ffffff; } */ /* 2.85:1 */
```

### ARIA Attributes

```tsx
// Interactive elements
function Dropdown({ label, isOpen, onToggle, children }: DropdownProps) {
  const listboxId = useId()
  return (
    <div>
      <button
        aria-haspopup="listbox"
        aria-expanded={isOpen}
        aria-controls={listboxId}
        onClick={onToggle}
      >
        {label}
      </button>
      <ul
        id={listboxId}
        role="listbox"
        aria-label={label}
        hidden={!isOpen}
      >
        {children}
      </ul>
    </div>
  )
}

// Progress / loading
<div
  role="progressbar"
  aria-valuenow={60}
  aria-valuemin={0}
  aria-valuemax={100}
  aria-label="Upload progress"
/>

// Live regions for dynamic updates
<div aria-live="polite" aria-atomic="true">
  {statusMessage}
</div>

// Form fields
<label htmlFor="email">Email address</label>
<input
  id="email"
  type="email"
  aria-describedby="email-hint email-error"
  aria-required="true"
  aria-invalid={!!errors.email}
/>
<p id="email-hint">We'll never share your email.</p>
{errors.email && <p id="email-error" role="alert">{errors.email}</p>}
```

### Skip Links

```tsx
// Skip to main content — essential for keyboard users
function SkipLink() {
  return (
    <a
      href="#main-content"
      className="sr-only focus:not-sr-only focus:fixed focus:top-4 focus:left-4
                 focus:z-50 focus:px-4 focus:py-2 focus:bg-blue-600 focus:text-white
                 focus:rounded-md focus:outline-none"
    >
      Skip to main content
    </a>
  )
}

// Usage in root layout
<body>
  <SkipLink />
  <Header />
  <main id="main-content" tabIndex={-1}>
    {children}
  </main>
</body>
```

### Focus Visible (Keyboard Users)

```css
/* Remove default outline but provide visible alternative for keyboard nav */
:focus { outline: none; }

:focus-visible {
  outline: 2px solid #3b82f6;
  outline-offset: 2px;
  border-radius: 2px;
}

/* Tailwind equivalent — add to component */
/* className="focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2" */
```

### Focus Trap in Modals

```tsx
import FocusTrap from 'focus-trap-react'

function Modal({ isOpen, onClose, children }: ModalProps) {
  useEffect(() => {
    if (isOpen) {
      const handleEscape = (e: KeyboardEvent) => {
        if (e.key === 'Escape') onClose()
      }
      document.addEventListener('keydown', handleEscape)
      return () => document.removeEventListener('keydown', handleEscape)
    }
  }, [isOpen, onClose])

  if (!isOpen) return null

  return (
    <FocusTrap focusTrapOptions={{ initialFocus: '#modal-close', returnFocusOnDeactivate: true }}>
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="modal-title"
        className="fixed inset-0 z-50 flex items-center justify-center"
      >
        <div className="bg-white rounded-xl p-6 max-w-md w-full">
          <h2 id="modal-title">Title</h2>
          {children}
          <button id="modal-close" onClick={onClose}>Close</button>
        </div>
      </div>
    </FocusTrap>
  )
}
```

### Semantic HTML

```tsx
// GOOD: Semantic structure
function PageLayout() {
  return (
    <>
      <header>
        <nav aria-label="Main navigation">
          <ul>
            <li><a href="/">Home</a></li>
            <li><a href="/about">About</a></li>
          </ul>
        </nav>
      </header>

      <main>
        <article>
          <header>
            <h1>Article Title</h1>
            <time dateTime="2026-04-02">April 2, 2026</time>
          </header>
          <section aria-labelledby="intro-heading">
            <h2 id="intro-heading">Introduction</h2>
            <p>Content...</p>
          </section>
        </article>

        <aside aria-label="Related articles">
          <h2>Related</h2>
        </aside>
      </main>

      <footer>
        <p><small>&copy; 2026 Company</small></p>
      </footer>
    </>
  )
}
```

### axe-core Testing

```ts
// Vitest / Jest integration
import { axe } from 'jest-axe'
import { render } from '@testing-library/react'

test('button has no a11y violations', async () => {
  const { container } = render(<Button>Click me</Button>)
  const results = await axe(container)
  expect(results).toHaveNoViolations()
})

// Storybook a11y addon — add to .storybook/main.ts
// import '@storybook/addon-a11y'
```

---

## SEO

### Next.js Metadata API

```tsx
// app/page.tsx — static metadata
export const metadata = {
  title: 'Home | My App',
  description: 'The best app for...',
  openGraph: {
    title: 'My App',
    description: 'The best app for...',
    url: 'https://myapp.com',
    siteName: 'My App',
    images: [{ url: 'https://myapp.com/og.png', width: 1200, height: 630 }],
    locale: 'en_US',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'My App',
    description: 'The best app for...',
    images: ['https://myapp.com/og.png'],
  },
}

// app/blog/[slug]/page.tsx — dynamic metadata
export async function generateMetadata({ params }: { params: { slug: string } }) {
  const post = await getPost(params.slug)
  if (!post) return { title: 'Not Found' }

  return {
    title: `${post.title} | Blog`,
    description: post.excerpt,
    openGraph: {
      title: post.title,
      description: post.excerpt,
      images: [{ url: post.coverImage, width: 1200, height: 630 }],
      type: 'article',
      publishedTime: post.publishedAt,
      authors: [post.author.name],
    },
  }
}
```

### JSON-LD Structured Data

```tsx
// app/layout.tsx — Organization schema
function OrganizationSchema() {
  const schema = {
    '@context': 'https://schema.org',
    '@type': 'Organization',
    name: 'My Company',
    url: 'https://myapp.com',
    logo: 'https://myapp.com/logo.png',
    sameAs: ['https://twitter.com/myapp', 'https://linkedin.com/company/myapp'],
  }
  return <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(schema) }} />
}

// app/products/[id]/page.tsx — Product schema
function ProductSchema({ product }: { product: Product }) {
  const schema = {
    '@context': 'https://schema.org',
    '@type': 'Product',
    name: product.name,
    description: product.description,
    image: product.imageUrl,
    offers: {
      '@type': 'Offer',
      price: product.price,
      priceCurrency: 'USD',
      availability: product.inStock
        ? 'https://schema.org/InStock'
        : 'https://schema.org/OutOfStock',
    },
  }
  return <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(schema) }} />
}

// app/blog/[slug]/page.tsx — Article schema
function ArticleSchema({ post }: { post: Post }) {
  const schema = {
    '@context': 'https://schema.org',
    '@type': 'Article',
    headline: post.title,
    datePublished: post.publishedAt,
    dateModified: post.updatedAt,
    author: { '@type': 'Person', name: post.author.name },
    image: post.coverImage,
  }
  return <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(schema) }} />
}
```

### Sitemap

```ts
// app/sitemap.ts
import { MetadataRoute } from 'next'

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const posts = await getAllPosts()

  const staticRoutes: MetadataRoute.Sitemap = [
    { url: 'https://myapp.com', lastModified: new Date(), changeFrequency: 'daily', priority: 1 },
    { url: 'https://myapp.com/about', lastModified: new Date(), changeFrequency: 'monthly', priority: 0.8 },
    { url: 'https://myapp.com/blog', lastModified: new Date(), changeFrequency: 'weekly', priority: 0.9 },
  ]

  const postRoutes: MetadataRoute.Sitemap = posts.map(post => ({
    url: `https://myapp.com/blog/${post.slug}`,
    lastModified: new Date(post.updatedAt),
    changeFrequency: 'weekly',
    priority: 0.7,
  }))

  return [...staticRoutes, ...postRoutes]
}
```

### robots.txt

```ts
// app/robots.ts
import { MetadataRoute } from 'next'

export default function robots(): MetadataRoute.Robots {
  return {
    rules: [
      { userAgent: '*', allow: '/', disallow: ['/api/', '/admin/'] },
    ],
    sitemap: 'https://myapp.com/sitemap.xml',
  }
}
```

---

## Key Rules

- Every image needs a meaningful `alt` attribute — empty string (`alt=""`) for decorative images.
- Keyboard navigation must work without a mouse: Tab → move, Enter/Space → activate, Escape → close.
- Use `useId()` to generate unique IDs for `htmlFor`/`aria-describedby` pairs in React.
- All form inputs need visible labels — `aria-label` is a last resort for icon-only controls.
- Test with a screen reader (NVDA/VoiceOver) and keyboard-only navigation before shipping.
- JSON-LD schemas should be validated with Google's Rich Results Test before deployment.
- Include canonical `<link>` tags to prevent duplicate content penalties.
