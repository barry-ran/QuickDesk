export function isWindowsPath(path) {
    return /^[A-Za-z]:/.test(path) || path.includes('\\');
}

export function separatorForPath(path) {
    return isWindowsPath(path) ? '\\' : '/';
}

function withoutTrailingSeparator(path) {
    if (!path || path === '/' || /^[A-Za-z]:\\?$/.test(path)) return path;
    return path.replace(/[\\/]+$/, '');
}

export function parentPath(path) {
    const clean = withoutTrailingSeparator(path || '');
    if (!clean) return '';
    if (clean === '/') return '/';
    if (/^[A-Za-z]:$/.test(clean)) return `${clean}\\`;
    if (/^[A-Za-z]:\\$/.test(clean)) return clean;

    const index = Math.max(clean.lastIndexOf('/'), clean.lastIndexOf('\\'));
    if (index < 0) return '';
    if (index === 0) return '/';
    if (index === 2 && isWindowsPath(clean)) return `${clean.slice(0, 2)}\\`;
    return clean.slice(0, index);
}

export function basename(path) {
    const clean = withoutTrailingSeparator(path || '');
    if (!clean || clean === '/' || /^[A-Za-z]:\\?$/.test(clean)) return '';
    const index = Math.max(clean.lastIndexOf('/'), clean.lastIndexOf('\\'));
    return index < 0 ? clean : clean.slice(index + 1);
}

export function joinRemotePath(directory, name) {
    if (!directory) return name;
    if (!name) return directory;
    const last = directory.at(-1);
    if (last === '/' || last === '\\') return directory + name;
    return `${directory}${separatorForPath(directory)}${name}`;
}

export function breadcrumbs(path) {
    const clean = withoutTrailingSeparator(path || '');
    if (!clean) return [];

    if (isWindowsPath(clean)) {
        const drive = clean.slice(0, 2);
        const rest = clean.slice(2).replace(/^[\\/]+/, '');
        const result = [{ label: drive, path: `${drive}\\` }];
        let current = `${drive}\\`;
        for (const part of rest.split(/[\\/]+/).filter(Boolean)) {
            current = joinRemotePath(current, part);
            result.push({ label: part, path: current });
        }
        return result;
    }

    const result = [{ label: '/', path: '/' }];
    let current = '';
    for (const part of clean.split('/').filter(Boolean)) {
        current += `/${part}`;
        result.push({ label: part, path: current });
    }
    return result;
}

export function normalizeEntries(entries) {
    if (!Array.isArray(entries)) return [];
    return entries
        .filter(entry => entry && typeof entry === 'object' && entry.name)
        .map(entry => {
            const type = entry.type === 'dir' || entry.type === 'directory'
                ? 'directory'
                : 'file';
            return {
                name: String(entry.name),
                type,
                size: Number(entry.size) || 0,
                isDirectory: type === 'directory',
            };
        })
        .sort((left, right) => {
            if (left.isDirectory !== right.isDirectory) {
                return left.isDirectory ? -1 : 1;
            }
            return left.name.localeCompare(right.name, undefined, { sensitivity: 'base' });
        });
}

export function validateEntryName(value) {
    const name = String(value ?? '').trim();
    if (!name) throw new Error('A name is required');
    if (name === '.' || name === '..') throw new Error('This name is invalid');
    if (/[\\/]/.test(name)) throw new Error('A name cannot contain a path separator');
    if (/[\u0000-\u001f]/.test(name)) throw new Error('A name cannot contain control characters');
    return name;
}

export function formatFileSize(bytes) {
    const value = Number(bytes) || 0;
    if (value < 1024) return `${value} B`;
    if (value < 1024 ** 2) return `${(value / 1024).toFixed(1)} KB`;
    if (value < 1024 ** 3) return `${(value / 1024 ** 2).toFixed(1)} MB`;
    return `${(value / 1024 ** 3).toFixed(2)} GB`;
}
