// @ts-check
import withNuxt from './.nuxt/eslint.config.mjs'
import betterTailwindcss from 'eslint-plugin-better-tailwindcss'
import { getDefaultAttributes } from 'eslint-plugin-better-tailwindcss/api/defaults'

export default withNuxt(
  betterTailwindcss.configs['correctness-error'],
  {
    settings: {
      'better-tailwindcss': {
        entryPoint: 'app/assets/css/main.css',
        attributes: [
          ...getDefaultAttributes(),
          ['^v-bind:ui$', [{ match: 'objectValues' }]]
        ]
      }
    }
  },
  {
    rules: {
      // Gaya proyek: atribut template boleh satu baris; overload defineEmits adalah idiom Vue.
      'vue/max-attributes-per-line': 'off',
      '@typescript-eslint/unified-signatures': 'off',
      '@stylistic/max-statements-per-line': 'off',
      // Utang lama yang butuh refactor bertahap: tetap terlihat sebagai peringatan, tidak memblokir commit.
      '@typescript-eslint/no-explicit-any': 'warn',
      'vue/no-mutating-props': 'warn',
      '@typescript-eslint/no-dynamic-delete': 'warn',
      // Kelas CSS buatan sendiri (style scoped / global), bukan utilitas Tailwind.
      'better-tailwindcss/no-unknown-classes': ['error', {
        ignore: ['^markdown-', '^office-', '^is-fullscreen$']
      }]
    }
  },
  {
    // Kantor Agent: pixel art berbasis kelas CSS scoped (o-*, pm-*, bagian tubuh/meja), bukan utilitas Tailwind.
    files: ['app/components/office/**'],
    rules: { 'better-tailwindcss/no-unknown-classes': 'off' }
  }
)
