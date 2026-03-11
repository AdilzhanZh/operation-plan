<template>
  <section class="not-found page">
    <div class="panel panel-warning not-found-card">
      <span class="kicker">404</span>
      <h2>Страница не найдена</h2>
      <p>
        Похоже, адрес больше не существует или вы перешли по устаревшей ссылке. Вернитесь в рабочий контур и продолжите работу оттуда.
      </p>

      <div class="not-found-actions">
        <RouterLink :to="{ name: 'dashboard' }" class="btn btn-primary">
          Вернуться на Dashboard
        </RouterLink>
      </div>
    </div>
  </section>
</template>

<style scoped>
.not-found {
  min-height: calc(100vh - 8rem);
  place-content: center;
}

.not-found-card {
  max-width: 42rem;
  padding: 2rem;
}

h2 {
  margin: 1rem 0 0;
  font-size: clamp(2rem, 5vw, 3rem);
  line-height: 0.96;
  letter-spacing: -0.05em;
}

p {
  margin: 1rem 0 0;
  color: var(--muted);
  max-width: 32rem;
}

.not-found-actions {
  margin-top: 1.5rem;
}
</style>
