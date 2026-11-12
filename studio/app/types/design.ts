/** Galeri Desain: referensi tampilan website untuk dianalisis AI & dirangkum jadi Design Brief. */

export type DesignStatus = 'new' | 'captured' | 'analyzed' | 'failed'
export type DesignSource = 'seed' | 'manual' | 'agent'
export type DesignAnalysisSource = 'llm_vision' | 'llm_text' | 'heuristic'
export type DesignPlatform = 'web' | 'android' | 'ios'
export type DesignPriority = 'must' | 'should' | 'nice'

export interface DesignShot {
  file: string
  label?: string
  mime: string
}

/** Satu best practice yang terlihat di screenshot & bisa diterapkan ke aplikasi lain. */
export interface DesignPrinciple {
  area: string
  rule: string
  evidence?: string
  how_to_apply?: string
  priority?: DesignPriority
}

export interface DesignColor {
  hex: string
  count?: number
  role?: string
  name?: string
  neutral?: boolean
}

export interface DesignValueCount {
  value: string
  count: number
}

export interface DesignTokens {
  colors: DesignColor[]
  css_variables?: DesignColor[]
  fonts: DesignValueCount[]
  heading_font?: string
  google_fonts?: string[]
  font_sizes?: DesignValueCount[]
  radii?: DesignValueCount[]
  shadows?: DesignValueCount[]
  breakpoints?: number[]
  frameworks?: string[]
  has_dark_mode: boolean
  uses_gradient: boolean
  uses_blur: boolean
  uses_grid: boolean
  uses_animation: boolean
  theme_color?: string
  css_bytes: number
  stylesheets: number
}

export interface PageStructure {
  headings: string[]
  nav_links: string[]
  ctas?: string[]
  counts: Record<string, number>
  text_length: number
  language?: string
}

/** Profil desain: cara desainer + frontend developer membaca sebuah tampilan (nilai baku backend, label dari /facets). */
export interface DesignFacets {
  purposes?: string[]
  goals?: string[]
  tones?: string[]
  audiences?: string[]
  archetypes?: string[]
  styles?: string[]
  density?: string
  scheme?: string
  /** dimensi yang ditetapkan kurator (tidak ditimpa analisis ulang) */
  pinned?: string[]
  source?: 'llm' | 'heuristic' | 'manual' | 'mixed'
}

export type DesignFacetDim = 'purposes' | 'goals' | 'tones' | 'audiences' | 'archetypes' | 'styles' | 'density' | 'scheme'

export interface DesignFacetItem { value: string, label: string, references: number }
export interface DesignFacetVocab { dims: Record<DesignFacetDim, DesignFacetItem[]>, order: DesignFacetDim[] }

/** Input facet kurator: array kosong = lepas pin; density/scheme '' = lepas pin. Dimensi yang tak dikirim tidak berubah. */
export interface DesignFacetsInput {
  purposes?: string[]
  goals?: string[]
  tones?: string[]
  audiences?: string[]
  archetypes?: string[]
  styles?: string[]
  density?: string
  scheme?: string
}

export interface DesignAnalysis {
  summary: string
  style_keywords: string[]
  mood?: string
  palette: DesignColor[]
  typography: { heading: string, body: string, mono?: string, scale?: string, notes?: string }
  layout: { grid?: string, spacing?: string, density?: string, patterns?: string[] }
  sections?: string[]
  components?: { name: string, description: string }[]
  effects?: string[]
  reuse_tips?: string[]
  tailwind_hint?: string
  agent_prompt?: string
  navigation?: { type?: string, placement?: string, shape?: string, items?: string, active_state?: string, secondary?: string, mobile_pattern?: string }
  spacing?: { base_unit?: string, scale?: string, section_padding?: string, component_padding?: string, gaps?: string, density?: string }
  forms?: { label_placement?: string, input_style?: string, input_height?: string, grouping?: string, action_placement?: string, validation?: string }
  buttons?: { primary?: string, secondary?: string, shape?: string, sizing?: string, placement?: string }
  color_usage?: { distribution?: string, accent_usage?: string, semantic?: string, contrast?: string }
  principles?: DesignPrinciple[]
}

export interface DesignReference {
  id: number
  name: string
  platform: DesignPlatform
  url: string
  category: string
  tags: string[]
  notes: string
  source: DesignSource
  rating: number
  favorite: boolean
  status: DesignStatus
  last_error?: string
  desktop_shot?: string
  mobile_shot?: string
  uploads: DesignShot[]
  captured_at?: string
  page_title?: string
  page_description?: string
  tokens?: DesignTokens
  structure?: PageStructure
  analysis?: DesignAnalysis
  facets?: DesignFacets
  analysis_source?: DesignAnalysisSource
  analysis_model?: string
  analyzed_at?: string
  created_at: string
  updated_at: string
  relevance?: number
  /** bacaan teori warna palet (hanya di detail GET /api/design-library/:id) */
  color_theory?: ColorTheory
}

export type ColorHarmony = 'neutral_accent' | 'achromatic' | 'monochromatic' | 'analogous' | 'complementary' | 'split_complementary' | 'triadic' | 'tetradic' | 'free'

export interface TheoryColor {
  hex: string
  role?: string
  hue: number
  saturation: number
  lightness: number
  neutral: boolean
}

/** Teori warna atas palet referensi: deskripsi (harmoni roda warna, suhu, kontras), tanpa skor. */
export interface ColorTheory {
  harmony: ColorHarmony
  harmony_label: string
  temperature: 'warm' | 'cool' | 'mixed' | 'neutral'
  temperature_label: string
  saturation?: 'muted' | 'balanced' | 'vivid'
  saturation_label?: string
  hues: number[]
  colors: TheoryColor[]
  text_contrast?: number
  text_hex?: string
  background_hex?: string
  /** id konsep teori terkait (node Pusat Pengetahuan theory:<id>) */
  concepts: string[]
  summary: string
}

export interface DesignTheoryExample {
  id: number
  name: string
  platform: DesignPlatform
  note?: string
  colors?: string[]
}

/** Konsep teori desain/warna + jembatannya ke Galeri Desain. */
export interface DesignTheoryEntry {
  id: string
  group: 'design' | 'color'
  title: string
  summary: string
  apply: string
  avoid: string
  areas: string[]
  tags: string[]
  harmony?: ColorHarmony
  node_key: string
  examples: DesignTheoryExample[]
  patterns?: string[]
  observed: number
}

export interface DesignLibraryResponse {
  references: DesignReference[]
  categories: string[]
  platforms: DesignPlatform[]
  keywords: DesignValueCount[]
  /** jumlah referensi per nilai facet: facets[dim][value] */
  facets?: Record<string, Record<string, number>>
  stats: { total: number, analyzed: number, pending: number, failed: number, favorites: number }
}

export interface DesignReferenceInput {
  name?: string
  platform?: DesignPlatform
  url?: string
  category?: string
  tags?: string[]
  notes?: string
  rating?: number
  favorite?: boolean
  facets?: DesignFacetsInput
}

export interface DesignBriefRequest {
  ids?: number[]
  query?: string
  category?: string
  limit?: number
  goal?: string
  project_id?: number
  write?: boolean
}

export interface DesignBrief {
  references: { id: number, name: string, url: string, category: string, keywords: string[], analyzed: boolean }[]
  style_keywords: DesignValueCount[]
  palette: DesignColor[]
  typography: DesignAnalysis['typography']
  patterns: DesignValueCount[]
  sections: string[]
  effects: string[]
  tips: string[]
  avoid: string[]
  css_variables: string
  principles: DesignPrincipleArea[]
  markdown: string
  written_to?: string
  generated_at: string
}

export interface DesignPrincipleEntry extends DesignPrinciple {
  sources: string[]
  count: number
  baseline: boolean
  /** id referensi sumber (kosong = prinsip umum bawaan) */
  source_ids?: number[]
}

export interface DesignPrincipleArea {
  area: string
  label: string
  principles: DesignPrincipleEntry[]
}

/** Basis prinsip desain: standar Material/HIG/WCAG + hasil ekstraksi screenshot referensi. */
export interface DesignKnowledgeBase {
  areas: DesignPrincipleArea[]
  total: number
  references: number
  markdown: string
}

export interface VisionModelCandidate {
  config_id: number
  config_name: string
  model: string
  active: boolean
  selected: boolean
  /** Hasil uji vision terakhir (gambar uji 4 kotak warna) sejak daemon jalan. */
  probe?: VisionProbeResult
}

export interface VisionProbeResult {
  ok: boolean
  model: string
  config?: string
  expected: string
  answer?: string
  error?: string
  latency_ms: number
  checked_at: string
}

/** Website → Halaman → Komponen: hasil UKUR browser (posisi, ukuran, spacing, style), bukan tebakan LLM. */
export interface MeasuredBox { x: number, y: number, w: number, h: number }

export interface MeasuredComponentStyle {
  font?: string
  font_size?: number
  font_weight?: number
  line_height?: number
  letter_spacing?: number
  text_transform?: string
  text_align?: string
  color?: string
  background?: string
  gradient?: boolean
  border?: string
  radius?: number
  shadow?: string
  display?: string
  contrast?: number
}

export type MeasuredRole = 'navbar' | 'heading' | 'button' | 'input' | 'card' | 'form' | 'image' | 'footer' | 'section' | 'text' | 'icon' | 'group' | 'item'

export interface MeasuredComponent {
  id: number
  role: MeasuredRole
  level?: number
  label?: string
  tag: string
  section: number
  /** ID komponen terukur terdekat yang membungkusnya (kosong = langsung di section) */
  parent?: number
  /** group: isi tiap butir yang berulang, mis. "card" atau "icon+heading+text" */
  item?: string
  box: MeasuredBox
  x_pct: number
  w_pct: number
  align?: string
  /** atas, kanan, bawah, kiri (px) */
  margin: [number, number, number, number]
  padding: [number, number, number, number]
  gap?: number
  children?: number
  columns?: number
  above_fold: boolean
  style: MeasuredComponentStyle
  signature: string
}

export interface MeasuredSection {
  index: number
  role: string
  /** jenis menurut isi: hero, features, cards, products, pricing, testimonials, faq, cta, … */
  kind?: string
  label?: string
  y: number
  h: number
  padding_top: number
  padding_bottom: number
  background?: string
  columns?: number
  components: number
}

export interface PageMeasure {
  /** 2 = komponen bertingkat (parent, group, item) + jenis section; kosong = daftar rata */
  version?: number
  viewport_w: number
  viewport_h: number
  page_height: number
  layout: {
    container_width: number
    container_left: number
    side_padding: number
    sections: MeasuredSection[]
    header_height?: number
    header_sticky?: boolean
    nav_items?: number
    h_overflow: boolean
    fold_cta: boolean
    fold_h1: boolean
    h1_count: number
    largest_text_is_h1: boolean
  }
  components: MeasuredComponent[]
  stats: {
    font_sizes: DesignValueCount[]
    spacing: DesignValueCount[]
    radii: DesignValueCount[]
    fonts: DesignValueCount[]
    body_size: number
    body_line_height: number
    h1_size: number
    chars_per_line: number
    text_samples: number
    low_contrast: number
    low_contrast_examples?: string[]
    interactive: number
    small_targets: number
  }
}

/** Fakta terukur satu halaman (apa adanya — tanpa target, skor, atau lulus/gagal). */
export interface ObservedFact {
  key: string
  area: string
  title: string
  viewport: 'desktop' | 'mobile'
  value: string
  num: number
  unit?: string
  detail?: string
}

export interface DesignPage {
  id: number
  reference_id: number
  url: string
  kind: string
  title?: string
  auto: boolean
  status: string
  last_error?: string
  measured_at?: string
  desktop_shot?: string
  mobile_shot?: string
  desktop?: PageMeasure
  mobile?: PageMeasure
  observations?: ObservedFact[]
}

export interface ComponentVariant {
  role: MeasuredRole
  signature: string
  count: number
  pages: string[]
  viewport: string
  sample: MeasuredComponent
  median_w: number
  median_h: number
}

export interface MeasuredNorms {
  pages: number
  container_width: number
  side_padding_mobile: number
  section_padding_desktop: number
  section_padding_mobile: number
  header_height: number
  body_size: number
  body_line_height: number
  h1_desktop: number
  h1_mobile: number
  button_height: number
  button_padding_x: number
  button_radius: number
  card_padding: number
  card_radius: number
  card_gap: number
  input_height: number
  nav_items: number
  chars_per_line: number
}

/** Median & rentang umum satu angka di jaringan pola (satu suara per website). */
export interface PatternNumber { median: number, low: number, high: number, sites: number }

export interface PatternExample { reference_id: number, name: string, page_id: number, page_kind: string, section: number }

/** Satu pola bentuk di jaringan: varian section, komposisi (isi), komponen, atau elemen. Tanpa warna. */
export interface PatternNode {
  id: string
  level: 'section' | 'composition' | 'component' | 'element'
  kind?: string
  key: string
  label: string
  sites: number
  uses: number
  contexts?: Record<string, number>
  numbers?: Record<string, PatternNumber>
  examples?: PatternExample[]
  sample?: string
}

export interface PatternLink {
  node: PatternNode
  sites: number
  in_context: number
  /** angka bentuk ini di dalam varian yang ditanyakan */
  numbers?: Record<string, PatternNumber>
}

export interface PatternAnswer {
  variant: PatternNode
  in_context: number
  compositions: PatternLink[]
  components: PatternLink[]
  elements: PatternLink[]
  with?: PatternLink[]
  next?: PatternLink[]
  before?: PatternLink[]
}

export interface PatternQuery { text?: string, kind?: string, surface?: string, layout?: string, purpose?: string, tone?: string, limit?: number }

export interface PatternResult {
  query: PatternQuery
  context?: string[]
  sites: number
  context_sites: number
  relaxed?: string[]
  answers: PatternAnswer[]
  markdown: string
}

export interface PatternKindSummary { kind: string, variants: number, sites: number, in_context: number }

export interface DesignPatternsResponse {
  result: PatternResult
  sites: number
  pages: number
  nodes: number
  edges: number
  kinds: PatternKindSummary[]
  /** urutan varian section halaman beranda yang paling sering */
  skeleton: PatternNode[] | null
}

export interface DesignPagesResponse {
  pages: DesignPage[]
  /** sari bentuk per ID halaman: susunan yang disarikan dari tampilan (desktop & mobile) + teks ringkasnya */
  shapes?: Record<string, { text: string, desktop: unknown, mobile: unknown }>
  components: { desktop: ComponentVariant[], mobile: ComponentVariant[] }
  norms: MeasuredNorms | null
  /** kesimpulan galeri untuk jenis website referensi ini (pembanding, bukan nilai) */
  conclusion: DesignConclusion | null
  kinds: string[]
}

/** Sebaran satu angka teramati lintas website sekelompok (satu suara per website). */
export interface ConclusionMetric {
  key: string
  label: string
  area: string
  unit?: string
  sites: number
  /** fakta ya/tidak: porsi website "ya" (0–1) */
  median: number
  low: number
  high: number
  min: number
  max: number
  share?: boolean
  yes?: number
}

export interface ConclusionTrait {
  key: string
  label: string
  sites: number
  values: { value: string, label: string, sites: number }[]
}

export interface ConclusionPattern {
  area: string
  rule: string
  how_to_apply?: string
  sites: number
  site_names?: string[]
}

/** Kesimpulan Galeri Desain untuk satu jenis website (purpose '' = semua jenis). */
export interface DesignConclusion {
  platform: string
  purpose: string
  label: string
  sites: number
  pages: number
  strength: 'awal' | 'sedang' | 'kuat'
  references: { id: number, name: string }[]
  metrics?: ConclusionMetric[]
  traits?: ConclusionTrait[]
  patterns?: ConclusionPattern[]
  statements: string[]
  /** data jenis ini cukup lengkap → mode galeri murni (pengetahuan desain bawaan LLM tidak dipakai) */
  sufficient: boolean
  /** yang masih kurang agar cukup */
  gaps?: string[]
  markdown: string
}

/** Tag desain satu aplikasi FE/mobile proyek — kosakata facet sama dengan referensi Galeri Desain. */
export interface AppDesignProfile {
  component_id: number
  project_id: number
  name: string
  role: string
  framework?: string
  platform: 'web' | 'android' | 'ios'
  platform_pinned?: boolean
  facets: DesignFacets
  labels?: string[]
  /** kesimpulan galeri yang berlaku untuk aplikasi ini */
  conclusion?: { purpose: string, label: string, sites: number, strength: 'awal' | 'sedang' | 'kuat', sufficient: boolean, gaps?: string[] }
}
