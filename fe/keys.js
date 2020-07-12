//. values to test against event.which
export const Escape = 27;
export const Enter = 13;

export function isEsc(ev) {
    return ev.which === Escape;
}

export function isEnter(ev) {
    return ev.which === Enter;
}

export function isUp(ev) {
    return (ev.key == "ArrowUp") || (ev.key == "Up");
}

export function isDown(ev) {
    return (ev.key == "ArrowDown") || (ev.key == "Down");
}

// navigation up is: Up or Ctrl-P
export function isNavUp(ev) {
    if (isUp(ev)) {
        return true;
    }
    return ev.ctrlKey && (ev.keyCode === 80);
}

// navigation down is: Down or Ctrl-N
export function isNavDown(ev) {
    if (isDown(ev)) {
        return true;
    }
    return ev.ctrlKey && (ev.keyCode === 78);
}
