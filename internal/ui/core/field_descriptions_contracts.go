package core

func init() {
	// ── CONTRACTS: DTOs ───────────────────────────────────────────────────────

	fieldDescriptions["category"] = map[string]string{
		"Request":       "Inbound payload DTO. Represents data the client sends to the API. Generates validation annotations and request binding.",
		"Response":      "Outbound payload DTO. Represents data the API returns to clients. Generates serialization code.",
		"Event Payload": "Message payload for async event systems. Generates schema definitions compatible with the selected serialization format.",
		"Shared/Common": "Reusable DTO referenced by multiple requests or responses. Generates a shared types module.",
	}

	fieldDescriptions["protocol"] = map[string]string{
		"REST/JSON":         "JSON over HTTP. Default for web APIs. Generates JSON struct tags and OpenAPI schema definitions.",
		"Protobuf":          "Binary Protocol Buffer encoding. Compact and fast. Generates .proto message definitions and compiled stubs.",
		"Avro":              "Binary Avro encoding. Schema registered in a schema registry. Generates Avro schema files and registry-aware serializer.",
		"MessagePack":       "Binary MessagePack encoding. JSON-compatible but more compact. Generates MessagePack codec wrappers.",
		"Thrift":            "Apache Thrift binary encoding. Multi-language RPC and serialization. Generates .thrift IDL files and language stubs.",
		"FlatBuffers":       "Zero-copy binary encoding. Extremely fast deserialization. Generates FlatBuffers schema and accessor code.",
		"Cap'n Proto":       "Zero-copy, schema-based binary encoding. No parse step needed. Generates Cap'n Proto schema and bindings.",
		"REST":              "Synchronous HTTP endpoint. Generates HTTP handler with method, path, and request/response types.",
		"GraphQL":           "Query language for APIs. Clients specify exact data needs. Generates resolver stubs and GraphQL schema types.",
		"gRPC":              "High-performance RPC over HTTP/2. Strong typing via Protobuf. Generates .proto service definitions and gRPC stubs.",
		"WebSocket message": "Bidirectional persistent connection. Real-time push and receive. Generates WebSocket handler with message dispatch.",
		"Event":             "Async message via a broker (Kafka, RabbitMQ). Decoupled delivery. Generates producer/consumer stubs and message schemas.",
		"WebSocket":         "Persistent bidirectional connection. Real-time server-push. Generates WebSocket upgrade handler.",
		"Webhook":           "Server pushes events to a registered client URL. Generates webhook dispatcher and HMAC signature validation.",
		"SOAP":              "XML-based web service protocol. Generates WSDL definition and SOAP envelope binding code.",
	}

	fieldDescriptions["http_method"] = map[string]string{
		"GET":    "Retrieve a resource. Idempotent and safe. Should not modify state.",
		"POST":   "Create a new resource or submit data. Not idempotent. Generates handler that persists a new record.",
		"PUT":    "Replace an entire resource. Idempotent. Generates handler that overwrites all fields of an existing record.",
		"PATCH":  "Partially update a resource. Generates handler that merges provided fields.",
		"DELETE": "Remove a resource. Idempotent. Generates handler that deletes or soft-deletes a record by ID.",
	}

	fieldDescriptions["graphql_op_type"] = map[string]string{
		"Query":        "Read-only data fetch. No side effects. Generates resolver that reads data.",
		"Mutation":     "Data modification operation. Creates, updates, or deletes. Generates resolver with input validation and persistence.",
		"Subscription": "Real-time data stream over WebSocket. Generates subscription resolver with event source wiring.",
	}

	fieldDescriptions["grpc_stream_type"] = map[string]string{
		"Unary":            "Single request, single response. Standard function call semantics. Default for most gRPC methods.",
		"Server stream":    "Single request, stream of responses from the server. Good for live feeds and large datasets.",
		"Client stream":    "Stream of requests from the client, single aggregated response. Good for batch uploads.",
		"Bidirectional":    "Both client and server stream simultaneously. Full-duplex. Used for chat and live collaboration.",
		"Server streaming": "Single request, stream of responses. Good for progress notifications.",
		"Client streaming": "Stream of requests, single response. Good for batch uploads or sensor data ingestion.",
	}

	fieldDescriptions["ws_direction"] = map[string]string{
		"Client→Server": "Messages flow from client to server only. Server processes incoming events.",
		"Server→Client": "Messages pushed from server to client only. Server-side broadcast or live updates.",
		"Bidirectional": "Both client and server send messages freely. Used for chat, collaborative editing, and real-time games.",
		"Send":          "This endpoint sends messages to connected clients.",
		"Receive":       "This endpoint receives messages from clients.",
	}

	fieldDescriptions["pagination"] = map[string]string{
		"Cursor-based": "Opaque cursor pointing to a result-set position. Consistent results during mutations. Best for real-time data.",
		"Offset/limit": "Standard page number and size. Simple to implement. Results may shift if data changes between pages.",
		"Keyset":       "Paginate by the last seen primary key. Efficient for large tables; no offset scanning.",
		"Page number":  "Client specifies page number and page size. Familiar UX. Same caveats as offset/limit.",
		"None":         "No pagination. Returns all results. Only suitable for small bounded result sets.",
	}

	fieldDescriptions["rate_limit"] = map[string]string{
		"Default (global)": "Uses the API gateway's global rate limit policy. No per-endpoint override.",
		"Strict":           "Lower limit than default. Extra protection for sensitive or expensive endpoints.",
		"Relaxed":          "Higher limit than default. For trusted clients or batch endpoints.",
		"None":             "No rate limiting on this endpoint.",
	}

	fieldDescriptions["deprecation"] = map[string]string{
		"None":                     "No deprecation notice. This API version is current.",
		"Sunset header":            "HTTP Sunset header added to responses announcing the removal date. RFC 8594 standard.",
		"Versioned removal notice": "Deprecation documented in API changelog with a specific removal version.",
		"Changelog entry":          "Deprecation noted in the project CHANGELOG only. No runtime header.",
		"Custom":                   "Custom deprecation strategy. Generates a placeholder for your deprecation logic.",
	}

	fieldDescriptions["tls_mode"] = map[string]string{
		"TLS":      "One-way TLS. Server presents certificate; client verifies it. Encrypts traffic.",
		"mTLS":     "Mutual TLS. Both client and server present certificates. Strong machine-to-machine authentication.",
		"Insecure": "No TLS. Plaintext connection. Only use in local development or fully trusted internal networks.",
	}

	fieldDescriptions["soap_version"] = map[string]string{
		"1.1": "SOAP 1.1 (HTTP+XML). Older standard; widely supported by legacy systems.",
		"1.2": "SOAP 1.2. Stricter, better defined semantics. Recommended for new SOAP integrations.",
	}

	fieldDescriptions["auth_mechanism"] = map[string]string{
		"API Key": "Static secret key sent in a header or query parameter. Generates API key validation middleware.",
		"OAuth2":  "OAuth 2.0 token-based auth. Generates OAuth2 client with token refresh logic.",
		"Bearer":  "JWT or opaque bearer token in the Authorization header. Generates token validation middleware.",
		"Basic":   "Base64-encoded username:password. Only use over HTTPS. Generates basic auth decoder.",
		"mTLS":    "Mutual TLS certificate authentication. Generates TLS config with client cert verification.",
		"None":    "No authentication on this external API. Suitable for public APIs.",
	}

	fieldDescriptions["failure_strategy"] = map[string]string{
		"Retry 3x":           "Retry failed requests up to three times with exponential backoff.",
		"Retry 5x":           "Retry failed requests up to five times. More aggressive recovery.",
		"Immediate fail":     "No retries. Return error on first failure. Application handles fallback.",
		"None":               "No explicit failure strategy. Framework defaults apply.",
		"Circuit breaker":    "Opens circuit after failure threshold, preventing further calls until it resets.",
		"Fallback":           "On failure, return a cached response or default value.",
		"Retry with backoff": "Retry with increasing delay between attempts. Reduces pressure on a struggling service.",
		"Timeout":            "Fail after a configured timeout. Prevents slow external calls from blocking the application.",
		"Timeout + fail":     "Apply a timeout; on expiry fail immediately without retries.",
	}

	// ── FRONTEND: Tech ────────────────────────────────────────────────────────

	fieldDescriptions["platform"] = map[string]string{
		// Frontend platforms
		"Web":     "Browser-based application delivered over HTTP. Generates a web project with HTML/CSS/JS output.",
		"Mobile":  "Native or cross-platform mobile application for iOS and/or Android.",
		"Desktop": "Native desktop application for macOS, Windows, or Linux.",
		"Hybrid":  "Targets multiple platforms from a single codebase.",
		// CI/CD platforms
		"GitHub Actions": "GitHub's native CI/CD. Tight repository integration. Generates workflow YAML files.",
		"GitLab CI":      "GitLab's built-in CI/CD with pipeline YAML. Generates .gitlab-ci.yml.",
		"Jenkins":        "Self-hosted CI/CD. Highly extensible via plugins. Generates Jenkinsfile.",
		"CircleCI":       "Cloud CI/CD with fast caching. Generates .circleci/config.yml.",
		"ArgoCD":         "GitOps continuous delivery for Kubernetes. Generates ArgoCD Application manifests.",
		"Tekton":         "Kubernetes-native CI/CD pipelines. Generates Tekton Pipeline and Task manifests.",
	}

	fieldDescriptions["meta_framework"] = map[string]string{
		"Next.js":           "React meta-framework. SSR, SSG, ISR, and file-based routing. Generates app router, API routes, and deployment config.",
		"Nuxt":              "Vue meta-framework. SSR, SSG, and file-based routing. Generates Nuxt config, composables, and server routes.",
		"SvelteKit":         "Svelte meta-framework. SSR and file-based routing with minimal JS. Generates SvelteKit routes and server-load functions.",
		"Remix":             "React meta-framework focused on web fundamentals. Nested routing and form actions. Generates loaders and actions.",
		"Astro":             "Islands architecture for content-heavy sites. Minimal client JS. Generates Astro pages with component islands.",
		"TanStack Start":    "React meta-framework from the TanStack team. Type-safe routing and server functions.",
		"Angular Universal": "Server-side rendering for Angular. Generates SSR server and TransferState setup.",
		"None":              "No meta-framework. Pure client-side rendering. Generates a SPA.",
	}

	fieldDescriptions["pkg_manager"] = map[string]string{
		"npm":  "Node's built-in package manager. Widest compatibility; default for most projects.",
		"yarn": "Facebook's package manager. Faster installs with lockfile determinism. Workspaces support.",
		"pnpm": "Disk-efficient package manager using a content-addressable store. Strictest dependency resolution.",
		"bun":  "All-in-one JS runtime and package manager. Fastest installs. Generates bun.lockb and bun-compatible scripts.",
	}

	fieldDescriptions["styling"] = map[string]string{
		"Tailwind CSS":      "Utility-first CSS framework. No custom CSS needed for most UIs. Generates Tailwind config and purge settings.",
		"CSS Modules":       "Locally scoped CSS. Each component has its own CSS file; classes are hashed. Generates module CSS files.",
		"Styled Components": "CSS-in-JS with tagged template literals. Dynamic styles based on props. Generates StyledComponent definitions.",
		"Sass/SCSS":         "CSS superset with variables, nesting, and mixins. Generates SCSS files with shared variables and partials.",
		"Vanilla CSS":       "Plain CSS. No framework or preprocessor. Full control; no added abstraction.",
		"UnoCSS":            "Atomic CSS engine. Faster than Tailwind with more flexibility. Generates UnoCSS preset config.",
	}

	fieldDescriptions["component_lib"] = map[string]string{
		"shadcn/ui":   "Unstyled, copy-paste components built on Radix UI. Full ownership of component code. Tailwind-based styling.",
		"Radix":       "Unstyled accessible primitives. Composable headless components for building custom design systems.",
		"Material UI": "Google Material Design React components. Comprehensive, opinionated. Generates MUI theme and component imports.",
		"Ant Design":  "Enterprise-grade React component library. Rich data table, form, and layout components.",
		"Headless UI": "Tailwind Labs headless components. Integrates with Tailwind CSS. Accessible by default.",
		"DaisyUI":     "Tailwind CSS component library with semantic class names. Generates DaisyUI theme config.",
		"None":        "No component library. Build UI components from scratch.",
		"Custom":      "Custom component library. Generates a component folder scaffold.",
	}

	fieldDescriptions["state_mgmt"] = map[string]string{
		"Redux Toolkit": "Opinionated Redux with reducers, actions, and thunks. Best for large apps with complex shared state.",
		"Zustand":       "Lightweight state management with hooks. Minimal boilerplate. Generates typed stores.",
		"Pinia":         "Vue's official state management. Composition API-based. Generates typed stores with actions and getters.",
		"MobX":          "Observable-based reactive state. Automatically tracks dependencies. Generates observable stores.",
		"Jotai":         "Atomic state management for React. Each atom is a unit of state. Generates typed atoms.",
		"Valtio":        "Proxy-based state with automatic re-renders. Minimal API. Generates state objects with snapshot reads.",
		"Context API":   "React's built-in context. Simple; no extra dependencies. Suitable for low-frequency state updates.",
		"XState":        "Finite state machine library. Explicit states and transitions. Generates machine definitions.",
		"NgRx":          "Redux-inspired state management for Angular. Actions, reducers, effects, and selectors.",
		"None":          "No dedicated state management library. Component-local state only.",
	}

	fieldDescriptions["data_fetching"] = map[string]string{
		"TanStack Query": "Async state management for server data. Caching, refetching, pagination, and optimistic updates.",
		"SWR":            "React hook for data fetching with stale-while-revalidate. Lightweight, automatic revalidation.",
		"Apollo Client":  "Feature-rich GraphQL client with normalized cache.",
		"Urql":           "Lightweight GraphQL client. Composable and extensible.",
		"RTK Query":      "Data-fetching layer built into Redux Toolkit. Cache invalidation tied to Redux state.",
		"Fetch API":      "Native browser fetch. No extra dependencies. Generates typed fetch wrappers.",
		"Axios":          "Promise-based HTTP client with interceptors. Generates Axios instance with base config.",
		"Vue Query":      "TanStack Query for Vue. Generates useQuery and useMutation composables.",
		"None":           "No data-fetching library. Raw fetch or XHR used directly.",
	}

	fieldDescriptions["form_handling"] = map[string]string{
		"React Hook Form": "Performant, flexible form library using uncontrolled inputs. Minimal re-renders. Generates form hooks and validation integration.",
		"Formik":          "Form state management with Yup validation integration. Higher-level API than React Hook Form.",
		"Zod + native":    "Form state managed by hand; validation via Zod schemas. No form library dependency.",
		"Vee-Validate":    "Vue form validation library with composition API support. Generates typed form validation composables.",
		"None":            "No form handling library. Forms managed with plain state.",
	}

	fieldDescriptions["validation"] = map[string]string{
		"Zod":             "TypeScript-first schema validation with type inference. Generates Zod schemas used for both runtime validation and TypeScript types.",
		"Yup":             "Schema-based object validation. Widely used with Formik. Generates Yup schema definitions.",
		"Valibot":         "Modular, tree-shakeable validation library. Smaller bundle than Zod. Generates Valibot schemas.",
		"Joi":             "Powerful validation for JavaScript objects. Battle-tested. Generates Joi schema definitions.",
		"Class-validator": "TypeScript decorator-based validation for classes. Best with NestJS or Angular. Generates decorated DTO classes.",
		"None":            "No validation library. Validation implemented manually.",
	}

	fieldDescriptions["realtime"] = map[string]string{
		"WebSocket": "Persistent bidirectional connection. Low latency, true push. Generates WebSocket client with reconnect logic.",
		"SSE":       "Server-Sent Events. Server pushes; client reads. Simpler than WebSocket for one-way streams.",
		"Polling":   "Periodic HTTP requests to check for updates. Simple; no persistent connection.",
		"None":      "No real-time data channel. Data refreshed on explicit user action only.",
	}

	fieldDescriptions["auth_flow"] = map[string]string{
		"Redirect (OAuth/OIDC)": "Redirect user to identity provider login page. Standard OAuth2/OIDC PKCE flow. Generates auth redirect handler, callback route, and token storage.",
		"Modal login":           "Login form shown in an overlay without leaving the current page.",
		"Magic link":            "Passwordless login via a one-time link emailed to the user.",
		"Passwordless":          "Login via OTP (email or SMS) without a password.",
		"Social only":           "Login exclusively via social providers (Google, GitHub). No username/password.",
	}

	fieldDescriptions["pwa_support"] = map[string]string{
		"None":                              "No PWA features. Standard web application.",
		"Basic (manifest + service worker)": "Web app manifest and basic service worker for install prompt and offline shell. Generates manifest.json and SW registration.",
		"Full offline":                      "Comprehensive offline-first PWA. Generates service worker with cache strategies (Workbox) for full offline functionality.",
		"Push notifications":                "Service worker with Web Push API. Generates push subscription management and notification display logic.",
	}

	fieldDescriptions["image_opt"] = map[string]string{
		"Next/Image (built-in)": "Next.js Image component with automatic resizing, lazy loading, and WebP conversion. Zero extra cost for Next.js projects.",
		"Cloudinary":            "Cloud-based image and video management. Generates Cloudinary SDK integration with transformation URL helpers.",
		"Imgix":                 "Real-time image processing CDN. URL-based transformations. Generates Imgix URL builder helpers.",
		"Sharp (self-hosted)":   "Node.js image processing library for server-side resizing and format conversion. Generates Sharp transform pipeline.",
		"CDN transform":         "Use CDN-native image transformation (Cloudflare Images, BunnyCDN). No application-level code needed.",
		"None":                  "No image optimization. Images served as-is.",
	}

	fieldDescriptions["error_boundary"] = map[string]string{
		"React Error Boundary": "React class component or react-error-boundary library. Catches rendering errors and shows fallback UI.",
		"Global try-catch":     "Top-level error handlers (window.onerror, unhandledrejection). Catches async errors outside the React tree.",
		"Framework default":    "Use the meta-framework's built-in error handling (Next.js error.tsx, SvelteKit +error.svelte).",
		"Custom":               "Custom error boundary implementation. Generates a typed ErrorBoundary component scaffold.",
	}

	fieldDescriptions["bundle_opt"] = map[string]string{
		"Code splitting (route-based)": "JavaScript split into per-route chunks loaded lazily. Fastest initial load. Generates route-based dynamic import configuration.",
		"Dynamic imports":              "On-demand loading of modules at runtime. Generates import() call patterns for heavy components.",
		"Tree shaking only":            "Dead code eliminated at build time. No lazy loading. Generates build config with side-effect annotations.",
		"None":                         "No bundle optimization. Single bundle output. Suitable for internal tools.",
	}

	// ── FRONTEND: Theme ───────────────────────────────────────────────────────

	fieldDescriptions["dark_mode"] = map[string]string{
		"None":                     "Light-only interface. No dark mode support.",
		"Toggle (user preference)": "User can switch between light and dark mode. Preference persisted in localStorage. Generates toggle component and CSS variable switching.",
		"System preference":        "Respects the OS dark mode setting via prefers-color-scheme. No manual toggle.",
		"Dark only":                "Dark-only interface. No light mode fallback.",
	}

	fieldDescriptions["border_radius"] = map[string]string{
		"Sharp (0)":     "Zero border radius. Hard-edged UI. Technical, developer-tool aesthetic.",
		"Subtle (4px)":  "Very slight rounding. Softens edges without looking rounded. Neutral; works across most design systems.",
		"Rounded (8px)": "Moderate rounding. Friendly and approachable. Popular in consumer SaaS.",
		"Pill (999px)":  "Fully rounded buttons and badges. Playful, modern aesthetic.",
		"Custom":        "Custom border-radius value. Specify your own design token.",
	}

	fieldDescriptions["spacing"] = map[string]string{
		"Compact":     "Tighter spacing scale. Fits more content on screen. Common in data-dense dashboards.",
		"Comfortable": "Balanced spacing. Default for most products.",
		"Spacious":    "Generous whitespace. Content breathes. Common in marketing and editorial layouts.",
	}

	fieldDescriptions["elevation"] = map[string]string{
		"Flat":      "No shadows or depth. Borders separate elements. Clean, minimal aesthetic.",
		"Subtle":    "Soft shadows for depth cues. Subtle layer separation without heavy drop shadows.",
		"Prominent": "Strong shadows and depth. Cards and modals clearly float above the background.",
	}

	fieldDescriptions["motion"] = map[string]string{
		"None":                   "No animations or transitions. Fastest perceived performance. Best for reduced-motion accessibility.",
		"Subtle transitions":     "Gentle opacity and translate transitions. Polished without distraction. 150-200ms ease transitions.",
		"Animated (spring/ease)": "Rich spring-physics or easing animations. Expressive interactions. Generates animation library config.",
	}

	fieldDescriptions["vibe"] = map[string]string{
		"Professional": "Clean, corporate aesthetic. Neutral palette. High information density. Suited for enterprise SaaS.",
		"Friendly":     "Warm colors, rounded elements, approachable typography. Suited for consumer apps.",
		"Playful":      "Bold colors, expressive animations, personality-forward. Suited for consumer products.",
		"Minimal":      "Extensive whitespace, limited color, restrained typography. Suited for content sites.",
		"Technical":    "Dark background, monospace accents, data-dense layout. Developer-facing tools.",
		"Custom":       "Custom vibe. Describe your design intent in the description field.",
	}

	fieldDescriptions["font"] = map[string]string{
		"Inter":   "Highly legible geometric sans-serif. Excellent screen rendering. Default for many design systems.",
		"Geist":   "Vercel's clean sans-serif. Optimized for developer tools and dashboards.",
		"DM Sans": "Rounded, friendly geometric sans-serif. Warm and modern.",
		"System":  "Platform system font stack. Zero download. Fastest loading.",
		"Custom":  "Custom font selection. Specify font-family, weights, and load strategy.",
	}

	// ── FRONTEND: Analytics ───────────────────────────────────────────────────

	fieldDescriptions["analytics"] = map[string]string{
		"PostHog":            "Open-source product analytics with feature flags, session recording, and funnel analysis.",
		"Google Analytics 4": "Google's event-based analytics platform. Deep integration with Google Ads. Generates GA4 gtag setup.",
		"Plausible":          "Privacy-first, lightweight analytics. GDPR-compliant. No cookies. Generates script snippet.",
		"Mixpanel":           "Event-based user analytics with cohort analysis. Generates Mixpanel SDK init and event helpers.",
		"Segment":            "Customer data platform. Routes events to multiple analytics destinations. Generates Segment analytics.js setup.",
		"Custom":             "Custom analytics integration. Generates typed event tracking helpers.",
		"None":               "No analytics configured.",
	}

	fieldDescriptions["telemetry"] = map[string]string{
		"Sentry":            "Error tracking and performance monitoring. Captures exceptions with full stack traces and context. Generates Sentry SDK init.",
		"Datadog RUM":       "Real User Monitoring from Datadog. Tracks page load, errors, and user interactions. Generates Datadog RUM init.",
		"LogRocket":         "Session replay with error tracking. Shows exactly what users experienced when errors occurred.",
		"New Relic Browser": "Full-stack observability with browser agent. Tracks page performance and JS errors.",
		"Custom":            "Custom frontend error tracking. Generates typed error reporter scaffold.",
		"None":              "No frontend RUM or error tracking.",
	}

	// ── FRONTEND: Navigation ──────────────────────────────────────────────────

	fieldDescriptions["nav_type"] = map[string]string{
		"Top bar":          "Horizontal navigation at the top of the viewport. Works well for apps with 5-10 top-level sections.",
		"Sidebar":          "Vertical navigation panel on the left. Scales to many sections. Common in dashboards.",
		"Bottom tabs":      "Tab bar at the bottom. Mobile-native pattern. Thumb-friendly on phones.",
		"Hamburger menu":   "Collapsed navigation behind a hamburger icon. Saves space on small screens.",
		"Breadcrumbs only": "Navigation expressed as a breadcrumb trail. No global nav. For deeply hierarchical content.",
		"None":             "No navigation component generated.",
	}

	fieldDescriptions["auth_aware"] = map[string]string{
		"true":  "Navigation dynamically shows or hides items based on authentication state. Generates auth-aware nav guards.",
		"false": "Navigation is static. Auth state not reflected in nav items.",
	}

	fieldDescriptions["breadcrumbs"] = map[string]string{
		"true":  "Breadcrumb trail generated above page content. Shows hierarchical path. Generates breadcrumb component.",
		"false": "No breadcrumbs.",
	}

	// ── FRONTEND: A11y/SEO ────────────────────────────────────────────────────

	fieldDescriptions["wcag_level"] = map[string]string{
		"A":    "Minimum WCAG compliance. Basic keyboard navigation and alt-text. Required for most public-sector sites.",
		"AA":   "Standard WCAG level. Color contrast, resize, and focus indicators. Recommended baseline for all products.",
		"AAA":  "Highest WCAG compliance. Extended audio description, sign language. Difficult to achieve fully.",
		"None": "No WCAG compliance target. Accessibility handled manually.",
	}

	fieldDescriptions["seo_render_strategy"] = map[string]string{
		"SSR":       "Server-Side Rendering. HTML generated on each request. Fresh content, crawlable. Generates SSR entry points.",
		"SSG":       "Static Site Generation. HTML pre-built at deploy time. Fastest load; no server needed. Generates static export config.",
		"ISR":       "Incremental Static Regeneration. Static pages rebuilt in background after revalidation period. Next.js-specific.",
		"Prerender": "Pre-renders specific routes to static HTML via a headless browser. Generates prerender configuration.",
		"None":      "No server-side or pre-rendering. Client-side SPA only.",
	}

	fieldDescriptions["sitemap"] = map[string]string{
		"true":  "Generates a sitemap.xml for search engine crawling. Configures sitemap route and robots.txt.",
		"false": "No sitemap generated.",
	}

	fieldDescriptions["meta_tag_injection"] = map[string]string{
		"true":  "Dynamic meta tags (title, description, og:image) injected per page. Generates meta tag management helpers.",
		"false": "Static meta tags only. No dynamic per-page meta injection.",
	}

	// ── FRONTEND: i18n ────────────────────────────────────────────────────────

	fieldDescriptions["translation_strategy"] = map[string]string{
		"i18n library": "Dedicated i18n library (next-intl, vue-i18n, i18next). Generates library config, translation files, and locale switching.",
		"Static files": "JSON translation files loaded statically. Simple; no runtime dependency on i18n library.",
		"CDN":          "Translation strings fetched from a CDN or localization platform (Lokalise, Phrase). Generates fetch-on-load logic.",
		"Custom":       "Custom translation mechanism. Generates a typed translation function scaffold.",
	}

	fieldDescriptions["timezone_handling"] = map[string]string{
		"Server-side":     "Timestamps stored and formatted on the server. Consistent across all clients.",
		"Client-side":     "Timestamps formatted in the user's local timezone by the browser.",
		"Both":            "Server normalizes to UTC; client formats to local timezone. Best of both worlds.",
		"UTC always":      "All dates displayed in UTC. No timezone conversion. Suitable for developer tools.",
		"User preference": "User sets their preferred timezone in profile settings.",
	}
}
