/**
 * Copyright 2023 Jack Bister
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { JSX, h } from "preact";
import {
  Anchor,
  AppShell,
  Flex,
  Group,
  Header,
  Navbar,
  MantineProvider,
  Text,
  ThemeIcon,
  Title,
  UnstyledButton,
  useMantineTheme,
} from "@mantine/core";
import { IconSearch, IconSettings } from "@tabler/icons-preact";

interface MainLinkProps {
  iconSvg: JSX.Element;
  href: string;
  label: string;
}

const MainLink = ({ iconSvg, href, label }: MainLinkProps) => {
  return (
    <UnstyledButton
      component="a"
      href={href}
      sx={(theme) => ({
        display: "block",
        width: "100%",
        padding: theme.spacing.xs,
        borderRadius: theme.radius.sm,
        color:
          theme.colorScheme === "dark" ? theme.colors.dark[0] : theme.black,

        "&:hover": {
          backgroundColor:
            theme.colorScheme === "dark"
              ? theme.colors.dark[6]
              : theme.colors.gray[0],
        },
      })}
    >
      <Group>
        <ThemeIcon variant="light">{iconSvg}</ThemeIcon>
        <Text size="sm">{label}</Text>
      </Group>
    </UnstyledButton>
  );
};

export const LogsuckAppShell = (props: any) => {
  const theme = useMantineTheme();
  return (
    <MantineProvider withGlobalStyles>
      <div id="app">
        <AppShell
          padding="md"
          navbar={
            <MantineProvider theme={{ colorScheme: "dark" }}>
              <Navbar width={{ base: 160 }} top={0}>
                <Navbar.Section>
                  <Title>
                    <Anchor href="/" px="xs">
                      Logsuck
                    </Anchor>
                  </Title>
                </Navbar.Section>
                <Navbar.Section>
                  <MainLink
                    label="Search"
                    iconSvg={<IconSearch />}
                    href="/search"
                  ></MainLink>
                  <MainLink
                    label="Config"
                    iconSvg={<IconSettings />}
                    href="/config"
                  ></MainLink>
                </Navbar.Section>
              </Navbar>
            </MantineProvider>
          }
          styles={(theme: any) => ({
            main: {
              backgroundColor:
                theme.colorScheme === "dark"
                  ? theme.colors.dark[8]
                  : theme.colors.gray[0],
            },
          })}
        >
          {props.children}
        </AppShell>
      </div>
    </MantineProvider>
  );
};
