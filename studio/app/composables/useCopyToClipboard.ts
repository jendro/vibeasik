// Salin teks ke clipboard + toast konfirmasi (gagal diam-diam, seperti salinan lokal sebelumnya).
export const useCopyToClipboard = () => {
  const toast = useToast()

  const copyToClipboard = async (text: string, label = 'Teks berhasil disalin!') => {
    try {
      await navigator.clipboard.writeText(text)
      toast.add({
        title: 'Disalin ke Clipboard',
        description: label,
        icon: 'i-lucide-clipboard-check',
        color: 'neutral'
      })
    } catch {
      // ignore
    }
  }

  return { copyToClipboard }
}
