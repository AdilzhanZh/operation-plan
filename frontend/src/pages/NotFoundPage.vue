<template>
  <section class="not-found">
    <h2>Page Not Found</h2>
    <RouterLink :to="{ name: 'dashboard' }">Return to dashboard</RouterLink>
  </section>
</template>

<style scoped>
.not-found {
  min-height: 50vh;
  display: grid;
  place-content: center;
  gap: 0.8rem;
}

h2 {
  margin: 0;
}
</style>
