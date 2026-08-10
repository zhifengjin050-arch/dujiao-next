import { getImageUrl } from './image'

/**
 * ? HTML ???????????????????
 * /uploads/xxx -> http://domain/uploads/xxx
 */
export function processHtmlForDisplay(html: string): string {
    if (!html) return ''

    let processed = html.replace(/style="([^"]*)"/g, (_match, styles: string) => {
        const cleaned = styles
            .replace(/(?:^|;)\s*color\s*:\s*[^;"]+/gi, '')
            .replace(/(?:^|;)\s*background-color\s*:\s*[^;"]+/gi, '')
            .replace(/(?:^|;)\s*background\s*:\s*[^;"]+/gi, '')
            .replace(/^\s*;\s*|\s*;\s*$/g, '')
            .replace(/;{2,}/g, ';')
            .trim()
        return cleaned ? `style="${cleaned}"` : ''
    })

    processed = processed.replace(/style='([^']*)'/g, (_match, styles: string) => {
        const cleaned = styles
            .replace(/(?:^|;)\s*color\s*:\s*[^;']+/gi, '')
            .replace(/(?:^|;)\s*background-color\s*:\s*[^;']+/gi, '')
            .replace(/(?:^|;)\s*background\s*:\s*[^;']+/gi, '')
            .replace(/^\s*;\s*|\s*;\s*$/g, '')
            .replace(/;{2,}/g, ';')
            .trim()
        return cleaned ? `style='${cleaned}'` : ''
    })

    return processed.replace(/src=["'](\/uploads\/.*?)["']/g, (_, path) => {
        return `src="${getImageUrl(path)}"`
    })
}

/**
 * ? HTML ???????????????????
 * http://domain/uploads/xxx -> /uploads/xxx
 */
export function processHtmlForStorage(html: string): string {
    if (!html) return ''

    const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || ''
    let apiHost = ''
    try {
        if (apiBaseUrl) {
            // ???? (?? localhost:8080 ? domain.com)
            apiHost = new URL(apiBaseUrl).host
        } else {
            // ????? API_BASE_URL,??????,??????
            apiHost = window.location.host
        }
    } catch (e) {
        // Fallback
        apiHost = window.location.host
    }

    // ?? src="value" ? src='value'
    return html.replace(/src=["'](.*?)["']/g, (match, src) => {
        try {
            // ????? URL (http:// ? https://)
            if (src.startsWith('http://') || src.startsWith('https://')) {
                const url = new URL(src)

                // ???????? API ??,???? /uploads/ ??
                // url.host ???????? (?? localhost:8080)
                if (url.host === apiHost && url.pathname.startsWith('/uploads/')) {
                    // ??????,?? src="/uploads/xxx.png"
                    // ?????????
                    return `src="${url.pathname}"`
                }
            }
        } catch (e) {
            // URL ????,??,????
        }
        return match
    })
}
