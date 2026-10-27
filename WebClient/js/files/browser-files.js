// Clipboard/drop APIs expose File objects only; never infer local filesystem paths.
export function filesFromDataTransfer(data) {
    const files = [];
    let directories = false;
    if (data?.items?.length) {
        for (const item of data.items) {
            if (item.kind !== 'file') continue;
            const entry = item.webkitGetAsEntry?.();
            if (entry?.isDirectory) { directories = true; continue; }
            const file = item.getAsFile?.();
            if (file) files.push(file);
        }
    } else {
        files.push(...Array.from(data?.files || []));
    }
    return { files, directories };
}
