export interface TechInfo {
  id: string
  name: string
  icon: string
  color: string
  category: 'language' | 'frontend' | 'backend' | 'mobile' | 'database' | 'orm' | 'auth' | 'tool' | 'ai' | 'other'
  description: string
}

export interface TechCategory {
  id: string
  label: string
  icon: string
  items: TechInfo[]
}

export const TECH_REGISTRY: Record<string, TechInfo> = {
  // ==================== PROGRAMMING LANGUAGES ====================
  go: {
    id: 'go',
    name: 'Go',
    icon: 'i-simple-icons-go',
    color: '#00ADD8',
    category: 'language',
    description: 'Bahasa pemrograman backend performa tinggi & concurency dari Google'
  },
  typescript: {
    id: 'typescript',
    name: 'TypeScript',
    icon: 'i-simple-icons-typescript',
    color: '#3178C6',
    category: 'language',
    description: 'Superset JavaScript bertipe statis untuk aplikasi web skala besar'
  },
  javascript: {
    id: 'javascript',
    name: 'JavaScript',
    icon: 'i-simple-icons-javascript',
    color: '#F7DF1E',
    category: 'language',
    description: 'Bahasa pemrograman dinamis standar web modern'
  },
  dart: {
    id: 'dart',
    name: 'Dart',
    icon: 'i-simple-icons-dart',
    color: '#0175C2',
    category: 'language',
    description: 'Bahasa pemrograman client-optimized untuk Flutter'
  },
  python: {
    id: 'python',
    name: 'Python',
    icon: 'i-simple-icons-python',
    color: '#3776AB',
    category: 'language',
    description: 'Bahasa serbaguna untuk backend, AI/ML, dan otomasi scripting'
  },
  php: {
    id: 'php',
    name: 'PHP',
    icon: 'i-simple-icons-php',
    color: '#777BB4',
    category: 'language',
    description: 'Bahasa scripting server-side populer untuk web & Laravel'
  },
  rust: {
    id: 'rust',
    name: 'Rust',
    icon: 'i-simple-icons-rust',
    color: '#DEA584',
    category: 'language',
    description: 'Sistem bahasa pemrograman ultra-cepat, memory-safe tanpa GC'
  },
  ruby: {
    id: 'ruby',
    name: 'Ruby',
    icon: 'i-simple-icons-ruby',
    color: '#CC342D',
    category: 'language',
    description: 'Bahasa dinamis berfokus pada produktivitas & Rails'
  },
  kotlin: {
    id: 'kotlin',
    name: 'Kotlin',
    icon: 'i-simple-icons-kotlin',
    color: '#7F52FF',
    category: 'language',
    description: 'Bahasa modern multiplatform dan standar utama Android'
  },
  swift: {
    id: 'swift',
    name: 'Swift',
    icon: 'i-simple-icons-swift',
    color: '#F05138',
    category: 'language',
    description: 'Bahasa pemrograman modern untuk ekosistem Apple (iOS & macOS)'
  },
  cplusplus: {
    id: 'cplusplus',
    name: 'C++',
    icon: 'i-simple-icons-cplusplus',
    color: '#00599C',
    category: 'language',
    description: 'Bahasa pemrograman performa tinggi untuk sistem dan game'
  },
  csharp: {
    id: 'csharp',
    name: 'C#',
    icon: 'i-simple-icons-csharp',
    color: '#239120',
    category: 'language',
    description: 'Bahasa berorientasi objek modern untuk .NET platform'
  },
  html: {
    id: 'html',
    name: 'HTML5',
    icon: 'i-simple-icons-html5',
    color: '#E34F26',
    category: 'language',
    description: 'Standar markup struktur konten web'
  },
  css: {
    id: 'css',
    name: 'CSS3',
    icon: 'i-simple-icons-css3',
    color: '#1572B6',
    category: 'language',
    description: 'Standar styling lembar gaya cascading web'
  },

  // ==================== FRONTEND & WEB FRAMEWORKS ====================
  nuxt: {
    id: 'nuxt',
    name: 'Nuxt',
    icon: 'i-simple-icons-nuxt',
    color: '#00DC82',
    category: 'frontend',
    description: 'Framework Vue intuitif untuk SSR, SPA, dan fullstack web modern'
  },
  vue: {
    id: 'vue',
    name: 'Vue.js',
    icon: 'i-simple-icons-vuedotjs',
    color: '#4FC08D',
    category: 'frontend',
    description: 'Framework JavaScript progresif untuk UI reaktif'
  },
  react: {
    id: 'react',
    name: 'React',
    icon: 'i-simple-icons-react',
    color: '#61DAFB',
    category: 'frontend',
    description: 'Library berbasis komponen untuk antarmuka pengguna'
  },
  next: {
    id: 'next',
    name: 'Next.js',
    icon: 'i-simple-icons-nextdotjs',
    color: '#000000',
    category: 'frontend',
    description: 'Framework React fullstack dengan Server Components dan App Router'
  },
  svelte: {
    id: 'svelte',
    name: 'Svelte',
    icon: 'i-simple-icons-svelte',
    color: '#FF3E00',
    category: 'frontend',
    description: 'UI compiler modern tanpa virtual DOM untuk kecepatan maksimal'
  },
  astro: {
    id: 'astro',
    name: 'Astro',
    icon: 'i-simple-icons-astro',
    color: '#BC52EE',
    category: 'frontend',
    description: 'Web framework berfokus pada konten dengan arsitektur Island'
  },
  angular: {
    id: 'angular',
    name: 'Angular',
    icon: 'i-simple-icons-angular',
    color: '#DD0031',
    category: 'frontend',
    description: 'Platform & framework enterprise-grade dari Google'
  },
  tailwindcss: {
    id: 'tailwindcss',
    name: 'Tailwind CSS',
    icon: 'i-simple-icons-tailwindcss',
    color: '#06B6D4',
    category: 'frontend',
    description: 'Utility-first CSS framework untuk pembuatan antarmuka cepat'
  },
  vite: {
    id: 'vite',
    name: 'Vite',
    icon: 'i-simple-icons-vite',
    color: '#646CFF',
    category: 'frontend',
    description: 'Frontend tooling & bundler kilat generasi baru'
  },

  // ==================== BACKEND FRAMEWORKS ====================
  fiber: {
    id: 'fiber',
    name: 'Fiber',
    icon: 'i-simple-icons-go',
    color: '#00ADD8',
    category: 'backend',
    description: 'Go web framework terinspirasi Express di atas Fasthttp'
  },
  gin: {
    id: 'gin',
    name: 'Gin',
    icon: 'i-simple-icons-gin',
    color: '#00ADD8',
    category: 'backend',
    description: 'HTTP web framework tercepat berbasis Radix Tree di Go'
  },
  echo: {
    id: 'echo',
    name: 'Echo',
    icon: 'i-simple-icons-go',
    color: '#00ADD8',
    category: 'backend',
    description: 'Framework HTTP mikro berkinerja tinggi untuk Go'
  },
  chi: {
    id: 'chi',
    name: 'Chi',
    icon: 'i-simple-icons-go',
    color: '#00ADD8',
    category: 'backend',
    description: 'Router modular ringan 100% kompatibel dengan net/http'
  },
  express: {
    id: 'express',
    name: 'Express',
    icon: 'i-simple-icons-express',
    color: '#000000',
    category: 'backend',
    description: 'Web framework minimalis & fleksibel untuk Node.js'
  },
  nestjs: {
    id: 'nestjs',
    name: 'NestJS',
    icon: 'i-simple-icons-nestjs',
    color: '#E0234E',
    category: 'backend',
    description: 'Framework progresif berbasis TypeScript dan modular arsitektur'
  },
  fastify: {
    id: 'fastify',
    name: 'Fastify',
    icon: 'i-simple-icons-fastify',
    color: '#000000',
    category: 'backend',
    description: 'Web framework berkecepatan tinggi dengan overhead rendah di Node.js'
  },
  hono: {
    id: 'hono',
    name: 'Hono',
    icon: 'i-simple-icons-hono',
    color: '#E36002',
    category: 'backend',
    description: 'Ultra-fast web framework untuk Cloudflare Workers, Deno, Bun, & Node'
  },
  fastapi: {
    id: 'fastapi',
    name: 'FastAPI',
    icon: 'i-simple-icons-fastapi',
    color: '#009688',
    category: 'backend',
    description: 'Framework Python modern berbasis async dan Pydantic types'
  },
  django: {
    id: 'django',
    name: 'Django',
    icon: 'i-simple-icons-django',
    color: '#092E20',
    category: 'backend',
    description: 'Framework Python batteries-included untuk pengembangan kilat'
  },
  flask: {
    id: 'flask',
    name: 'Flask',
    icon: 'i-simple-icons-flask',
    color: '#000000',
    category: 'backend',
    description: 'Mikro framework Python ringan dan mudah dikustomisasi'
  },
  laravel: {
    id: 'laravel',
    name: 'Laravel',
    icon: 'i-simple-icons-laravel',
    color: '#FF2D20',
    category: 'backend',
    description: 'PHP web framework elegan dengan ekosistem lengkap'
  },
  codeigniter: {
    id: 'codeigniter',
    name: 'CodeIgniter',
    icon: 'i-simple-icons-codeigniter',
    color: '#EF4223',
    category: 'backend',
    description: 'PHP MVC framework tangguh, ringan & berkinerja tinggi'
  },
  codeigniter4: {
    id: 'codeigniter4',
    name: 'CodeIgniter 4',
    icon: 'i-simple-icons-codeigniter',
    color: '#EF4223',
    category: 'backend',
    description: 'Framework PHP MVC modern dengan Spark CLI & routing filters'
  },
  symfony: {
    id: 'symfony',
    name: 'Symfony',
    icon: 'i-simple-icons-symfony',
    color: '#000000',
    category: 'backend',
    description: 'Kumpulan komponen reusable dan framework PHP tangguh'
  },
  springboot: {
    id: 'springboot',
    name: 'Spring Boot',
    icon: 'i-simple-icons-springboot',
    color: '#6DB33F',
    category: 'backend',
    description: 'Framework enterprise Java untuk aplikasi stand-alone siap produksi'
  },

  // ==================== MOBILE & CLIENT ====================
  flutter: {
    id: 'flutter',
    name: 'Flutter',
    icon: 'i-simple-icons-flutter',
    color: '#02569B',
    category: 'mobile',
    description: 'UI toolkit multiplatform dari Google untuk mobile, web, dan desktop'
  },
  android: {
    id: 'android',
    name: 'Android',
    icon: 'i-simple-icons-android',
    color: '#3DDC84',
    category: 'mobile',
    description: 'Sistem operasi mobile berbasis kernel Linux'
  },
  apple: {
    id: 'apple',
    name: 'iOS / Apple',
    icon: 'i-simple-icons-apple',
    color: '#000000',
    category: 'mobile',
    description: 'Ekosistem perangkat Apple iOS, iPadOS, dan macOS'
  },

  // ==================== DATABASES & STORAGE ====================
  postgresql: {
    id: 'postgresql',
    name: 'PostgreSQL',
    icon: 'i-simple-icons-postgresql',
    color: '#4169E1',
    category: 'database',
    description: 'Database relasional objek open-source terkemuka di dunia'
  },
  mysql: {
    id: 'mysql',
    name: 'MySQL',
    icon: 'i-simple-icons-mysql',
    color: '#4479A1',
    category: 'database',
    description: 'Sistem manajemen database SQL relasional yang luas digunakan'
  },
  mariadb: {
    id: 'mariadb',
    name: 'MariaDB',
    icon: 'i-simple-icons-mariadb',
    color: '#003545',
    category: 'database',
    description: 'Fork open source komunitas dari MySQL yang tangguh'
  },
  sqlite: {
    id: 'sqlite',
    name: 'SQLite',
    icon: 'i-simple-icons-sqlite',
    color: '#003B57',
    category: 'database',
    description: 'Mesin database SQL mandiri tanpa server dalam satu file'
  },
  redis: {
    id: 'redis',
    name: 'Redis',
    icon: 'i-simple-icons-redis',
    color: '#DC382D',
    category: 'database',
    description: 'Struktur data in-memory cepat untuk cache, broker, dan session'
  },
  mongodb: {
    id: 'mongodb',
    name: 'MongoDB',
    icon: 'i-simple-icons-mongodb',
    color: '#47A248',
    category: 'database',
    description: 'Database dokumen NoSQL fleksibel berbasis skema BSON'
  },
  supabase: {
    id: 'supabase',
    name: 'Supabase',
    icon: 'i-simple-icons-supabase',
    color: '#3ECF8E',
    category: 'database',
    description: 'Alternatif Firebase open source berbasis PostgreSQL penuh'
  },
  firebase: {
    id: 'firebase',
    name: 'Firebase',
    icon: 'i-simple-icons-firebase',
    color: '#FFCA28',
    category: 'database',
    description: 'Platform backend terkelola dari Google (Firestore, Auth, Storage)'
  },

  // ==================== ORM & DATA ACCESS ====================
  gorm: {
    id: 'gorm',
    name: 'GORM',
    icon: 'i-simple-icons-go',
    color: '#00ADD8',
    category: 'orm',
    description: 'ORM ramah pengembang untuk ekosistem Go'
  },
  prisma: {
    id: 'prisma',
    name: 'Prisma',
    icon: 'i-simple-icons-prisma',
    color: '#2D3748',
    category: 'orm',
    description: 'ORM generasi baru dengan skema deklaratif dan auto-migrasi'
  },
  drizzle: {
    id: 'drizzle',
    name: 'Drizzle ORM',
    icon: 'i-simple-icons-drizzle',
    color: '#C5F74F',
    category: 'orm',
    description: 'ORM TypeScript ringan tanpa overhead, SQL-like query builder'
  },
  typeorm: {
    id: 'typeorm',
    name: 'TypeORM',
    icon: 'i-simple-icons-typeorm',
    color: '#FE0803',
    category: 'orm',
    description: 'ORM berbasis decorator untuk TypeScript dan JavaScript'
  },

  // ==================== AUTH & SECURITY ====================
  jwt: {
    id: 'jwt',
    name: 'JWT',
    icon: 'i-simple-icons-jsonwebtokens',
    color: '#D63AFF',
    category: 'auth',
    description: 'JSON Web Tokens untuk stateless token-based authentication'
  },
  google_auth: {
    id: 'google_auth',
    name: 'Google OAuth',
    icon: 'i-simple-icons-google',
    color: '#4285F4',
    category: 'auth',
    description: 'Autentikasi Single Sign-On menggunakan akun Google'
  },
  auth0: {
    id: 'auth0',
    name: 'Auth0',
    icon: 'i-simple-icons-auth0',
    color: '#EB5424',
    category: 'auth',
    description: 'Platform manajemen identitas dan autentikasi aman'
  },
  betterauth: {
    id: 'betterauth',
    name: 'Better Auth',
    icon: 'i-simple-icons-betterauth',
    color: '#000000',
    category: 'auth',
    description: 'Framework autentikasi TypeScript komprehensif modern'
  },

  // ==================== PACKAGE MANAGERS & TOOLS ====================
  bun: {
    id: 'bun',
    name: 'Bun',
    icon: 'i-simple-icons-bun',
    color: '#F472B6',
    category: 'tool',
    description: 'All-in-one JavaScript runtime, bundler, dan package manager kilat'
  },
  pnpm: {
    id: 'pnpm',
    name: 'pnpm',
    icon: 'i-simple-icons-pnpm',
    color: '#F69220',
    category: 'tool',
    description: 'Package manager hemat disk dengan struktur hard link'
  },
  npm: {
    id: 'npm',
    name: 'npm',
    icon: 'i-simple-icons-npm',
    color: '#CB3837',
    category: 'tool',
    description: 'Registri paket dan manajer default ekosistem Node.js'
  },
  yarn: {
    id: 'yarn',
    name: 'Yarn',
    icon: 'i-simple-icons-yarn',
    color: '#2C8EBB',
    category: 'tool',
    description: 'Package manager handal untuk ekosistem JavaScript'
  },
  docker: {
    id: 'docker',
    name: 'Docker',
    icon: 'i-simple-icons-docker',
    color: '#2496ED',
    category: 'tool',
    description: 'Platform kontainerisasi untuk standardisasi deployment aplikasi'
  },
  git: {
    id: 'git',
    name: 'Git',
    icon: 'i-simple-icons-git',
    color: '#F05032',
    category: 'tool',
    description: 'Distributed version control system standar industri'
  },
  github: {
    id: 'github',
    name: 'GitHub',
    icon: 'i-simple-icons-github',
    color: '#24292F',
    category: 'tool',
    description: 'Platform kolaborasi kode dan version control berbasis Git'
  },
  composer: {
    id: 'composer',
    name: 'Composer',
    icon: 'i-simple-icons-composer',
    color: '#885630',
    category: 'tool',
    description: 'Dependency manager untuk ekosistem bahasa PHP'
  },

  // ==================== AI & LLM SERVICES ====================
  systemone: {
    id: 'systemone',
    name: 'System One',
    icon: 'i-lucide-cpu',
    color: '#10B981',
    category: 'ai',
    description: 'TypeSafe Jev AI Engine dengan model Laya untuk penalaran lokal'
  },
  openai: {
    id: 'openai',
    name: 'OpenAI',
    icon: 'i-simple-icons-openai',
    color: '#10A37F',
    category: 'ai',
    description: 'Model GPT-4o, reasoning, dan integrasi API OpenAI'
  },
  anthropic: {
    id: 'anthropic',
    name: 'Anthropic',
    icon: 'i-simple-icons-anthropic',
    color: '#D97757',
    category: 'ai',
    description: 'Model Claude 3.5 Sonnet / Opus untuk reasoning mendalam'
  },
  gemini: {
    id: 'gemini',
    name: 'Google Gemini',
    icon: 'i-simple-icons-google',
    color: '#4285F4',
    category: 'ai',
    description: 'Model multimodal canggih dari Google DeepMind'
  }
}

// Synonyms and normalization mapping
const ALIAS_MAP: Record<string, string> = {
  // Go
  'golang': 'go',

  // TS / JS
  'ts': 'typescript',
  'js': 'javascript',
  'node.js': 'javascript',
  'nodejs': 'javascript',
  'node': 'javascript',

  // Python
  'py': 'python',

  // PHP
  'php8': 'php',
  'php7': 'php',

  // Rust
  'rs': 'rust',

  // Frameworks
  'nuxt.js': 'nuxt',
  'nuxt3': 'nuxt',
  'nuxt 3': 'nuxt',
  'vue.js': 'vue',
  'vuejs': 'vue',
  'vue3': 'vue',
  'vue 3': 'vue',
  'react.js': 'react',
  'reactjs': 'react',
  'next.js': 'next',
  'nextjs': 'next',
  'tailwind': 'tailwindcss',
  'gofiber': 'fiber',
  'ci': 'codeigniter',
  'ci4': 'codeigniter4',
  'codeigniter': 'codeigniter',
  'codeigniter4': 'codeigniter4',
  'codeigniter 4': 'codeigniter4',

  // DB
  'postgres': 'postgresql',
  'psql': 'postgresql',
  'pg': 'postgresql',
  'sqlite3': 'sqlite',
  'mongo': 'mongodb',
  'firestore': 'firebase',

  // Auth
  'json web token': 'jwt',
  'jwt bearer token': 'jwt',
  'jwt token': 'jwt',
  'oauth': 'google_auth',
  'oauth2': 'google_auth',
  'oauth2 google': 'google_auth',
  'google oauth': 'google_auth',
  'firebase auth': 'firebase',
  'better-auth': 'betterauth',

  // Tools
  'cargo': 'rust',
  'pub': 'dart',
  'pip': 'python',

  // AI
  'jev': 'systemone',
  'laya': 'systemone',
  'system one': 'systemone',
  'system-one': 'systemone',
  'chatgpt': 'openai',
  'claude': 'anthropic'
}

/**
 * Resolves any tech keyword, language, or framework name into a rich TechInfo object.
 */
export const resolveTech = (raw: string | null | undefined): TechInfo => {
  if (!raw || typeof raw !== 'string') {
    return {
      id: 'unknown',
      name: 'Umum',
      icon: 'i-lucide-box',
      color: '#6B7280',
      category: 'other',
      description: 'Layanan atau modul umum'
    }
  }

  const cleaned = raw.trim().toLowerCase()

  // Exact registry check
  const exact = TECH_REGISTRY[cleaned]
  if (exact) {
    return exact
  }

  // Check alias map
  const mappedKey = ALIAS_MAP[cleaned]
  if (mappedKey) {
    const aliasMatch = TECH_REGISTRY[mappedKey]
    if (aliasMatch) {
      return aliasMatch
    }
  }

  // Partial / fuzzy match inside keyword
  for (const [alias, targetKey] of Object.entries(ALIAS_MAP)) {
    if (cleaned.includes(alias)) {
      const fuzzyMatch = TECH_REGISTRY[targetKey]
      if (fuzzyMatch) {
        return fuzzyMatch
      }
    }
  }

  for (const [key, info] of Object.entries(TECH_REGISTRY)) {
    if (cleaned.includes(key) && info) {
      return info
    }
  }

  // Fallback heuristic based on common hints
  if (cleaned.includes('db') || cleaned.includes('sql') || cleaned.includes('data')) {
    return {
      id: cleaned,
      name: raw,
      icon: 'i-lucide-database',
      color: '#0284C7',
      category: 'database',
      description: `Database engine: ${raw}`
    }
  }

  if (cleaned.includes('auth') || cleaned.includes('token') || cleaned.includes('pass') || cleaned.includes('key')) {
    return {
      id: cleaned,
      name: raw,
      icon: 'i-lucide-shield-check',
      color: '#8B5CF6',
      category: 'auth',
      description: `Metode otorisasi: ${raw}`
    }
  }

  if (cleaned.includes('api') || cleaned.includes('server') || cleaned.includes('http')) {
    return {
      id: cleaned,
      name: raw,
      icon: 'i-lucide-server',
      color: '#10B981',
      category: 'backend',
      description: `Layanan API / Server: ${raw}`
    }
  }

  return {
    id: cleaned,
    name: raw,
    icon: 'i-lucide-code',
    color: '#64748B',
    category: 'other',
    description: raw
  }
}

/**
 * Returns list of technologies grouped by category for the catalog view.
 */
export const getTechCategories = (): TechCategory[] => {
  const categoryDefs = [
    { id: 'language', label: 'Bahasa Pemrograman', icon: 'i-lucide-code' },
    { id: 'frontend', label: 'Frontend & Fullstack Web', icon: 'i-lucide-globe' },
    { id: 'backend', label: 'Backend Frameworks', icon: 'i-lucide-server' },
    { id: 'mobile', label: 'Mobile & Client OS', icon: 'i-lucide-smartphone' },
    { id: 'database', label: 'Database & Storage', icon: 'i-lucide-database' },
    { id: 'orm', label: 'ORM & Query Builder', icon: 'i-lucide-workflow' },
    { id: 'auth', label: 'Autentikasi & Keamanan', icon: 'i-lucide-shield-check' },
    { id: 'tool', label: 'Package Manager & Tools', icon: 'i-lucide-wrench' },
    { id: 'ai', label: 'AI & LLM Engines', icon: 'i-lucide-brain' }
  ]

  const items = Object.values(TECH_REGISTRY)

  return categoryDefs.map(cat => ({
    id: cat.id,
    label: cat.label,
    icon: cat.icon,
    items: items.filter(t => t.category === cat.id)
  }))
}

export const useTechLogo = () => {
  return {
    resolveTech,
    TECH_REGISTRY,
    getTechCategories
  }
}
