FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

# System build + runtime dependencies (cached separately from Qt below).
RUN apt-get update && apt-get install -y --no-install-recommends \
      build-essential cmake ninja-build git ca-certificates curl zip unzip file \
      python3 python3-pip \
      libgl1-mesa-dev libxkbcommon-dev libglib2.0-dev libdbus-1-3 \
      libfontconfig1 libfreetype6 libx11-6 libxext6 libxrender1 libxcb1 \
      libxcb-cursor0 libxcb-icccm4 libxcb-image0 libxcb-keysyms1 libxcb-randr0 \
      libxcb-render-util0 libxcb-shape0 libxcb-sync1 libxcb-xfixes0 libxcb-xkb1 \
      libxkbcommon-x11-0 libsm6 libice6 \
      libgstreamer1.0-0 libgstreamer-plugins-base1.0-0 \
      libpulse-dev libasound2-dev \
      libnss3 libnspr4 libcairo2 libpango-1.0-0 libgtk-3-0t64 \
      libxdamage1 libxfixes3 libxrandr2 libxtst6 libxcomposite1 libxcursor1 \
      libxi6 libxinerama1 libxkbcommon0 libdrm2 libgbm1 libasound2t64 libpulse0 \
      libatk1.0-0t64 libatk-bridge2.0-0t64 libcups2t64 libatspi2.0-0t64 \
      libxshmfence1 libx11-xcb1 libxcb-dri3-0 libegl1 libgl1 libgles2 \
      libwayland-client0 libwayland-cursor0 libwayland-egl1 libglib2.0-0t64 \
      libfuse2t64 desktop-file-utils \
    && rm -rf /var/lib/apt/lists/*

# Qt 6.8.3 (matches upstream CI). Separate layer so apt changes don't re-download it.
RUN pip install --break-system-packages --no-cache-dir aqtinstall \
    && aqt install-qt linux desktop 6.8.3 linux_gcc_64 \
         -m qtmultimedia qtwebsockets --outputdir /opt/Qt

ENV Qt6_DIR=/opt/Qt/6.8.3/gcc_64
ENV CMAKE_PREFIX_PATH=/opt/Qt/6.8.3/gcc_64
ENV PATH=/opt/Qt/6.8.3/gcc_64/bin:$PATH