import http from 'k6/http';
import { check, sleep, group } from 'k6';

// КОНФИГУРАЦИЯ ТЕСТА
export const options = {
  executor: 'ramping-vus',
  startVUs: 0,
  stages: [
    { duration: '30s', target: 10 }, // Прогрев
    { duration: '1m', target: 100 }, // Этап 1: Легкая нагрузка
    { duration: '5m', target: 400 }, // Этап 2: Основная нагрузка 
    { duration: '10m', target: 400 }, // Этап 3: Пиковая нагрузка
    { duration: '1m', target: 0 },  // Остывание
  ],

  thresholds: {
    'http_req_duration': [
      {
        threshold: 'p(95)<200', // Условие: p(95) < 200ms
        abortOnFail: false,
        delayAbortEval: '10s',
      },
    ],
    'http_req_failed': [
      {
        threshold: 'rate<0.01', // Доля ошибок < 1%
        abortOnFail: false,
        delayAbortEval: '10s',
      }
    ],
  },
};


// ГЛОБАЛЬНЫЕ ПЕРЕМЕННЫЕ
const BASE_URL = 'http://app:8080';
const ADMIN_TOKEN = 'admin-secret-token';
const HEADERS = {
  'Content-Type': 'application/json',
  'Authorization': `Bearer ${ADMIN_TOKEN}`,
};

// SETUP
export function setup() {
  const teams = [];
  const users = [];
  for (let i = 1; i <= 50; i++) {
    const teamName = `load-test-team-${i}`;
    const members = [];
    for (let j = 1; j <= 100; j++) {
      const userId = `load-user-${i}-${j}`;
      members.push({ user_id: userId, username: `User ${i}-${j}`, is_active: true });
      users.push(userId);
    }

    const res = http.post(`${BASE_URL}/team/add`, JSON.stringify({ team_name: teamName, members }), { headers: HEADERS });
    check(res, { 'Команда успешно создана': (r) => r.status === 201 });
    if (res.status === 201) {
      teams.push(teamName);
    }
  }
  
  let prCount = 0;
  for (let i = 0; i < Math.min(50, users.length); i++) {
    const user = users[i];
    const prId = `setup-pr-${i}`;
    
    const res = http.post(
      `${BASE_URL}/pullRequest/create`,
      JSON.stringify({pull_request_id: prId, pull_request_name: `Setup PR ${i}`, author_id: user,}), { headers: HEADERS });
    
    if (res.status === 201) {
      prCount++;
    }
  }

  return { userIds: users, teams };
}

// ОСНОВНОЙ СЦЕНАРИЙ
export default function (data) {
  const randomUser = data.userIds[Math.floor(Math.random() * data.userIds.length)];
  const prId = `pr-${__VU}-${__ITER}`;

  if (Math.random() < 0.8) {
    group('Pull Request', function () {
      let assignedReviewers = [];

      group('Создание PR', function () {
        const payload = JSON.stringify({
          pull_request_id: prId,
          pull_request_name: `PR от ${randomUser}`,
          author_id: randomUser,
        });

        const res = http.post(`${BASE_URL}/pullRequest/create`, payload, { headers: HEADERS });
        check(res, { 'PR создан (статус 201)': (r) => r.status === 201 });
        if (res.status === 201) {
          assignedReviewers = res.json().pr.assigned_reviewers;
        }
      });

      sleep(1);

      group('Получение списка PR на ревью', function () {
        const anotherRandomUser = data.userIds[Math.floor(Math.random() * data.userIds.length)];
        const res = http.get(`${BASE_URL}/users/getReview?user_id=${anotherRandomUser}`, { headers: HEADERS });
        check(res, { 'Список ревью получен (статус 200)': (r) => r.status === 200 });
      });

      sleep(1);

      if (assignedReviewers.length > 0 && Math.random() < 0.7) {
        group('Переназначение ревьювера', function () {
          const oldUserId = assignedReviewers[0];
          const payload = JSON.stringify({ pull_request_id: prId, old_user_id: oldUserId });
          
          const res = http.post(`${BASE_URL}/pullRequest/reassign`, payload, { headers: HEADERS });
          
          check(res, { 'Ревьювер переназначен (200 или 409)': (r) => [200, 409].includes(r.status)});
        });

        sleep(0.5);
      }

      if (assignedReviewers.length > 0) {
        group('Слияние PR', function () {
          const payload = JSON.stringify({ pull_request_id: prId });
          const res = http.post(`${BASE_URL}/pullRequest/merge`, payload, { headers: HEADERS });
          check(res, { 'PR смержен (статус 200)': (r) => r.status === 200 });
        });
      }
    });
  } else {
    group('Массовая деактивация пользователей', function () {
      const teamIndex = Math.floor(Math.random() * data.teams.length);
      const teamName = data.teams[teamIndex];
      
      const teamUsers = data.userIds.filter(id => id.startsWith(`load-user-${teamIndex + 1}-`));
      const usersToDeactivate = teamUsers.slice(0, Math.min(3, teamUsers.length));
      
      const deactivateReq = {
        team_name: teamName,
        user_ids: usersToDeactivate
      };
      
      const deactivateRes = http.post(
        `${BASE_URL}/team/deactivateUsers`,
        JSON.stringify(deactivateReq),
        { headers: HEADERS }
      );
      
      check(deactivateRes, {
        'Пользователи деактивированы (статус 200)': (r) => r.status === 200,
        'Возвращены деактивированные пользователи': (r) => {
          if (r.status === 200) {
            const body = r.json();
            return body.deactivated_users && body.deactivated_users.length > 0;
          }
          return false;
        },
        'Время выполнения < 100ms': (r) => r.timings.duration < 100
      });
      
      if (deactivateRes.status !== 200) {
        console.error(`Ошибка деактивации: ${deactivateRes.status} - ${deactivateRes.body}`);
      }
    });
  }
  
  sleep(0.1);
}