export type PageCategory = 'proyek' | 'operasional' | 'ai' | 'sistem'

export interface PageScreenshotItem {
  id: string
  title: string
  subtitle: string
  route: string
  category: PageCategory
  categoryLabel: string
  description: string
  icon: string
  badge?: string
  screenshotPath: string
  fallbackGradient: string
  features: string[]
}

export interface ScreenshotCaptureOptions {
  fileName?: string
  quality?: number
  scale?: number
}
