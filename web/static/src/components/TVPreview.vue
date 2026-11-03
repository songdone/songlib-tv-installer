<template>
  <div class="sl-tv" @mouseenter="pause" @mouseleave="resume" @focusin="pause" @focusout="resume">
    <div class="sl-tv-frame">
      <img
        v-for="(shot, i) in shots"
        :key="shot.src"
        :src="shot.src"
        :alt="$t(shot.captionKey)"
        class="sl-tv-shot"
        :class="{ 'sl-tv-shot--active': i === current }"
        :aria-hidden="i !== current"
        :loading="i === 0 ? (eager ? 'eager' : 'lazy') : 'lazy'"
      />
    </div>
    <div v-if="shots.length > 1" class="sl-tv-bar">
      <p class="sl-caption" aria-live="polite">{{ $t(shots[current].captionKey) }}</p>
      <div class="sl-tv-dots" role="group">
        <button
          v-for="(shot, i) in shots"
          :key="'dot-' + i"
          type="button"
          class="sl-tv-dot"
          :class="{ 'sl-tv-dot--active': i === current }"
          :aria-label="$t(shot.captionKey)"
          :aria-pressed="i === current"
          @click="show(i)"
        ></button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from "vue";

// Real, already-published SongLib TV screenshots (marketing/tv/PROVENANCE.md).
// Never AI-generated mockups; the 16:9 frame uses object-fit: contain so
// lyrics and UI text are never cropped.
const props = defineProps({
  shots: {
    type: Array,
    required: true,
  },
  eager: {
    type: Boolean,
    default: false,
  },
  // Seconds between slides; 0 disables auto-advance.
  interval: {
    type: Number,
    default: 6,
  },
});

const current = ref(0);
let timer = null;

function show(i) {
  current.value = i;
  restart();
}

function advance() {
  current.value = (current.value + 1) % props.shots.length;
}

function stop() {
  if (timer) {
    clearInterval(timer);
    timer = null;
  }
}

function restart() {
  stop();
  start();
}

function start() {
  if (timer || props.interval <= 0 || props.shots.length < 2) return;
  // Respect reduced motion: keep a single static frame.
  if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
  timer = setInterval(advance, props.interval * 1000);
}

function pause() {
  stop();
}

function resume() {
  start();
}

onMounted(start);
onUnmounted(stop);
</script>

<style lang="postcss" scoped>
.sl-tv {
  min-width: 0;
}

.sl-tv-frame {
  position: relative;
  aspect-ratio: 16 / 9;
  border-radius: var(--sl-radius-md);
  overflow: hidden;
  border: 1px solid var(--sl-border);
  background: var(--sl-brand-ink);
  box-shadow: var(--sl-shadow-popover);
}

.sl-tv-shot {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: contain;
  opacity: 0;
  transition: opacity 380ms ease;
}

.sl-tv-shot--active {
  opacity: 1;
}

@media (prefers-reduced-motion: reduce) {
  .sl-tv-shot {
    transition: none;
  }
}

.sl-tv-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 10px;
}

.sl-tv-dots {
  display: inline-flex;
  gap: 7px;
}

.sl-tv-dot {
  width: 9px;
  height: 9px;
  padding: 0;
  border: 0;
  border-radius: 999px;
  background: var(--sl-border);
  cursor: pointer;
  transition: background-color var(--sl-motion-fast) ease, width var(--sl-motion-fast) ease;
}

.sl-tv-dot--active {
  width: 22px;
  background: var(--sl-gold);
}
</style>
